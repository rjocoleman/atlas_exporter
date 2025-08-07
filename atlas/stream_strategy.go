// SPDX-License-Identifier: LGPL-3.0-or-later

package atlas

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/czerwonk/atlas_exporter/exporter"
	"github.com/czerwonk/atlas_exporter/probe"

	"github.com/DNS-OARC/ripeatlas/measurement"
	"github.com/czerwonk/atlas_exporter/config"
	log "github.com/sirupsen/logrus"
)

type streamingStrategy struct {
	measurements   map[string]*exporter.Measurement
	cfg            *config.Config
	mu             sync.Mutex
}

// NewStreamingStrategy returns an strategy using the RIPE Atlas Streaming API
func NewStreamingStrategy(ctx context.Context, cfg *config.Config, bufferSize uint) Strategy {
	s := &streamingStrategy{
		cfg:          cfg,
		measurements: make(map[string]*exporter.Measurement),
	}

	s.start(ctx, cfg.Measurements, bufferSize)
	return s
}

func (s *streamingStrategy) start(ctx context.Context, measurements []config.Measurement, bufferSize uint) {
	resultCh := make(chan *measurement.Result, int(bufferSize))

	for _, m := range measurements {
		w := &streamStrategyWorker{
			resultCh:    resultCh,
			measurement: m,
		}
		go w.run(ctx)
	}

	go s.processMeasurementResults(resultCh)
}

func (s *streamingStrategy) processMeasurementResults(resultCh chan *measurement.Result) {
	for r := range resultCh {
		s.processMeasurementResult(r)
	}
}

func (s *streamingStrategy) processMeasurementResult(r *measurement.Result) {
	log.Infof("Got result for %d from probe %d", r.MsmId(), r.PrbId())

	probe, err := probeForID(r.PrbId())
	if err != nil {
		log.Error(err)
		return
	}

	s.add(r, probe)
}

func (s *streamingStrategy) add(m *measurement.Result, probe *probe.Probe) {
	s.mu.Lock()
	defer s.mu.Unlock()

	msm := strconv.Itoa(m.MsmId())

	mes, found := s.measurements[msm]
	if !found {
		var err error
		mes, err = measurementForType(m.Type(), msm, strconv.Itoa(m.Af()), s.cfg)
		if err != nil {
			log.Error(err)
			return
		}

		s.measurements[msm] = mes
	}

	mes.Add(m, probe)
}

func (s *streamingStrategy) MeasurementResults(ctx context.Context, ids []string) ([]*exporter.Measurement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]*exporter.Measurement, 0)
	for _, id := range ids {
		m, found := s.measurements[id]
		if !found {
			continue
		}

		result = append(result, m)
	}

	return result, nil
}
