// SPDX-License-Identifier: LGPL-3.0-or-later

package ntp

import (
	"github.com/czerwonk/atlas_exporter/config"
	"github.com/czerwonk/atlas_exporter/exporter"
)

const (
	ns  = "atlas"
	sub = "ntp"
)

// NewMeasurement returns a new instance of `exorter.Measurement` for a NTP measurement
func NewMeasurement(id string, cfg *config.Config) *exporter.Measurement {
	opts := []exporter.MeasurementOpt{}

	if cfg.FilterInvalidResults {
		opts = append(opts, exporter.WithValidator(&exporter.DefaultResultValidator{}))
	}

	if cfg.MaxResultAge > 0 {
		opts = append(opts, exporter.WithMaxResultAge(cfg.MaxResultAge))
	}

	return exporter.NewMeasurement(&ntpExporter{id}, opts...)
}
