// SPDX-License-Identifier: LGPL-3.0-or-later

package atlas

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/czerwonk/atlas_exporter/exporter"
	"github.com/czerwonk/atlas_exporter/probe"

	"github.com/DNS-OARC/ripeatlas/measurement"
	"github.com/czerwonk/atlas_exporter/config"
	log "github.com/sirupsen/logrus"
)

type streamingStrategy struct {
	measurements     map[string]*exporter.Measurement
	cfg              *config.Config
	mu               sync.Mutex
	connectedWorkers int32
	lastDataTime     int64
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
			strategy:    s,
		}
		go func() {
			if err := w.run(ctx); err != nil {
				log.Errorf("Worker error for measurement %s: %v", m.ID, err)
			}
		}()
	}

	go s.processMeasurementResults(resultCh)
}

func (s *streamingStrategy) processMeasurementResults(resultCh chan *measurement.Result) {
	for {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("Panic in measurement processor, restarting: %v", r)
				}
			}()

			// Process measurements until panic or channel closes
			for r := range resultCh {
				s.processMeasurementResult(r)
			}
		}()

		// Check if channel is actually closed vs panic recovery
		select {
		case _, ok := <-resultCh:
			if !ok {
				log.Info("Measurement result channel closed, exiting processor")
				return
			}
		default:
			// Was a panic, continue outer loop to restart processing
			time.Sleep(1 * time.Second) // Brief pause before restart
		}
	}
}

func (s *streamingStrategy) processMeasurementResult(r *measurement.Result) {
	log.Infof("Got result for %d from probe %d", r.MsmId(), r.PrbId())

	// Update last data time
	now := time.Now().Unix()
	atomic.StoreInt64(&s.lastDataTime, now)

	// Update metrics
	measurementID := strconv.Itoa(r.MsmId())
	LastDataTimestampGauge.WithLabelValues(measurementID).Set(float64(now))

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
	log.Debugf("Adding result for measurement ID '%s' (raw MsmId: %d)", msm, m.MsmId())

	mes, found := s.measurements[msm]
	if !found {
		log.Debugf("Creating new measurement object for ID '%s' of type '%s'", msm, m.Type())
		var err error
		mes, err = measurementForType(m.Type(), msm, strconv.Itoa(m.Af()), s.cfg)
		if err != nil {
			log.Error(err)
			return
		}

		s.measurements[msm] = mes
		log.Debugf("Stored measurement ID '%s' in map. Map now has keys: %v", msm, s.getMapKeys())
	}

	mes.Add(m, probe)
}

func (s *streamingStrategy) getMapKeys() []string {
	keys := make([]string, 0, len(s.measurements))
	for k := range s.measurements {
		keys = append(keys, k)
	}
	return keys
}

func (s *streamingStrategy) MeasurementResults(ctx context.Context, ids []string) ([]*exporter.Measurement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Debugf("MeasurementResults: Looking for IDs %v in map with keys %v", ids, s.getMapKeys())

	result := make([]*exporter.Measurement, 0)
	for _, id := range ids {
		m, found := s.measurements[id]
		if !found {
			log.Debugf("MeasurementResults: ID '%s' not found in measurements map", id)
			continue
		}
		log.Debugf("MeasurementResults: Found measurement for ID '%s'", id)
		result = append(result, m)
	}

	return result, nil
}

func (s *streamingStrategy) IsHealthy() bool {
	// Check if we have at least one connected worker
	connected := atomic.LoadInt32(&s.connectedWorkers)
	if connected <= 0 {
		log.Debug("Health check failed: no connected workers")
		return false
	}

	// If max data age is configured, also check data freshness
	if s.cfg.HealthMaxDataAge > 0 {
		lastData := atomic.LoadInt64(&s.lastDataTime)
		if lastData == 0 {
			log.Debug("Health check failed: no data received yet")
			return false
		}

		age := time.Since(time.Unix(lastData, 0))
		if age > s.cfg.HealthMaxDataAge {
			log.Debugf("Health check failed: data age %v exceeds max %v", age, s.cfg.HealthMaxDataAge)
			return false
		}
	}

	return true
}
