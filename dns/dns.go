// SPDX-License-Identifier: LGPL-3.0-or-later

package dns

import (
	"github.com/czerwonk/atlas_exporter/config"
	"github.com/czerwonk/atlas_exporter/exporter"
)

const (
	ns  = "atlas"
	sub = "dns"
)

// NewMeasurement returns a new instance of `exorter.Measurement` for a DNS measurement
func NewMeasurement(id, ipVersion string, cfg *config.Config) *exporter.Measurement {
	opts := []exporter.MeasurementOpt{
		exporter.WithHistograms(newRttHistogram(id, ipVersion, cfg.HistogramBuckets.DNS.Rtt)),
	}

	if cfg.FilterInvalidResults {
		opts = append(opts, exporter.WithValidator(&exporter.DefaultResultValidator{}))
	}

	if cfg.MaxResultAge > 0 {
		opts = append(opts, exporter.WithMaxResultAge(cfg.MaxResultAge))
	}

	return exporter.NewMeasurement(&dnsExporter{id: id, nsidEnabled: cfg.DNS.NSIDEnabled}, opts...)
}
