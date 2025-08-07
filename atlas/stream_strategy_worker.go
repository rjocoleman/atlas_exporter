// SPDX-License-Identifier: LGPL-3.0-or-later

package atlas

import (
	"context"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/DNS-OARC/ripeatlas"
	"github.com/DNS-OARC/ripeatlas/measurement"
	"github.com/czerwonk/atlas_exporter/config"
	log "github.com/sirupsen/logrus"
)

const (
	minRetryDelay = 1 * time.Second
	maxRetryDelay = 60 * time.Second
)

type streamStrategyWorker struct {
	resultCh      chan<- *measurement.Result
	measurement   config.Measurement
	retryAttempt  int
}

// getRetryDelay calculates exponential backoff with jitter
func (w *streamStrategyWorker) getRetryDelay() time.Duration {
	// Exponential: 1s, 2s, 4s, 8s, 16s, 32s, 60s (capped)
	delay := minRetryDelay * time.Duration(1<<uint(w.retryAttempt))
	if delay > maxRetryDelay {
		delay = maxRetryDelay
	}
	
	// Add jitter (±25%) to prevent thundering herd
	jitter := time.Duration(rand.Int63n(int64(delay / 2)))
	finalDelay := delay + jitter - (delay / 4)
	
	log.Debugf("Reconnection attempt %d for measurement #%s, waiting %v", 
		w.retryAttempt+1, w.measurement.ID, finalDelay)
	
	return finalDelay
}

func (w *streamStrategyWorker) run(ctx context.Context) error {
	for {
		ch, err := w.subscribe()
		if err != nil {
			log.Error(err)
			w.retryAttempt++
		} else {
			log.Infof("Subscribed to results of measurement #%s", w.measurement.ID)
			w.retryAttempt = 0 // Reset on successful connection
			w.listenForResults(ctx, ch)
			w.retryAttempt++ // Increment for next reconnection attempt
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(w.getRetryDelay()):
			continue
		}
	}
}

func (w *streamStrategyWorker) subscribe() (<-chan *measurement.Result, error) {
	stream := ripeatlas.NewStream()

	msm, err := strconv.Atoi(w.measurement.ID)
	if err != nil {
		return nil, err
	}

	ch, err := stream.MeasurementResults(ripeatlas.Params{
		"msm": msm,
	})
	if err != nil {
		return nil, err
	}

	return ch, nil
}

func (w *streamStrategyWorker) listenForResults(ctx context.Context, ch <-chan *measurement.Result) {
	for {
		select {
		case m, ok := <-ch:
			if !ok {
				log.Warnf("Stream closed for measurement #%s", w.measurement.ID)
				return
			}
			if m == nil {
				continue
			}

			if m.ParseError != nil {
				log.Error(m.ParseError)
			}

			if m.ParseError != nil && strings.HasPrefix(m.ParseError.Error(), "c.On(disconnect)") {
				log.Error(m.ParseError)
				return
			}

			w.resultCh <- m
		case <-ctx.Done():
			return
		}
	}
}
