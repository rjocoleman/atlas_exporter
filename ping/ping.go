// SPDX-License-Identifier: LGPL-3.0-or-later

package ping

import (
	"github.com/czerwonk/atlas_exporter/config"
	"github.com/czerwonk/atlas_exporter/exporter"
)

const (
	ns  = "atlas"
	sub = "ping"
)

// NewMeasurement returns a new instance of `exorter.Measurement` for a ping measurement
func NewMeasurement(id, ipVersion string, cfg *config.Config) *exporter.Measurement {
	opts := []exporter.MeasurementOpt{
		exporter.WithHistograms(newRttHistogram(id, ipVersion, cfg.HistogramBuckets.Ping.Rtt)),
	}

	if cfg.FilterInvalidResults {
		opts = append(opts, exporter.WithValidator(&exporter.DefaultResultValidator{}))
	}

	if cfg.MaxResultAge > 0 {
		opts = append(opts, exporter.WithMaxResultAge(cfg.MaxResultAge))
	}

	return exporter.NewMeasurement(&pingExporter{id}, opts...)
}
