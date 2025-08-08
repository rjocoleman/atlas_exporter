// SPDX-License-Identifier: LGPL-3.0-or-later

package exporter

import (
	"sync"
	"time"

	"github.com/DNS-OARC/ripeatlas/measurement"
	"github.com/czerwonk/atlas_exporter/probe"
	"github.com/prometheus/client_golang/prometheus"
)

// MeasurementOpt are options to apply to the `Measurement`
type MeasurementOpt func(r *Measurement)

// WithHistograms adds histograms to the measurement
func WithHistograms(h ...Histogram) MeasurementOpt {
	return func(r *Measurement) {
		r.histograms = append(r.histograms, h...)
	}
}

// WithValidator sets an validator to validate results for a measurement
func WithValidator(v ResultValidator) MeasurementOpt {
	return func(r *Measurement) {
		r.validator = v
	}
}

// WithMaxResultAge sets the maximum age for results to be included
func WithMaxResultAge(age time.Duration) MeasurementOpt {
	return func(r *Measurement) {
		r.maxResultAge = age
	}
}

// Measurement handles measurement results and converts to metrics
type Measurement struct {
	mu           sync.RWMutex
	latest       map[int]*measurement.Result
	probes       map[int]*probe.Probe
	histograms   []Histogram
	exporter     Exporter
	validator    ResultValidator
	maxResultAge time.Duration
}

// NewMeasurement returns a new instance of `Measurement`
func NewMeasurement(exporter Exporter, opts ...MeasurementOpt) *Measurement {
	r := &Measurement{
		latest:     make(map[int]*measurement.Result),
		probes:     make(map[int]*probe.Probe),
		histograms: make([]Histogram, 0),
		exporter:   exporter,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Add adds an result to a measurement
func (r *Measurement) Add(m *measurement.Result, probe *probe.Probe) {
	if r.validator != nil && !r.validator.IsValid(m, probe) {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.latest[m.PrbId()] = m
	r.probes[m.PrbId()] = probe

	for _, h := range r.histograms {
		h.ProcessResult(m)
	}
}

// Describe describes all metrics for the `Measurement`
func (r *Measurement) Describe(ch chan<- *prometheus.Desc) {
	r.exporter.Describe(ch)

	for _, h := range r.histograms {
		h.Hist().Describe(ch)
	}
}

// Collect collects metrics for the `Measurement`
func (r *Measurement) Collect(ch chan<- prometheus.Metric) {
	r.mu.RLock()
	// snapshot keys to avoid holding lock while exporting
	results := make([]*measurement.Result, 0, len(r.latest))
	for _, v := range r.latest {
		results = append(results, v)
	}
	probes := make(map[int]*probe.Probe, len(r.probes))
	for k, v := range r.probes {
		probes[k] = v
	}
	r.mu.RUnlock()

	for _, v := range results {
		// Skip stale results if max age is configured
		if r.maxResultAge > 0 {
			cutoff := time.Now().Unix() - int64(r.maxResultAge.Seconds())
			if v.Timestamp() < int(cutoff) {
				continue
			}
		}
		r.exporter.Export(v, probes[v.PrbId()], ch)
	}

	for _, h := range r.histograms {
		h.Hist().Collect(ch)
	}
}
