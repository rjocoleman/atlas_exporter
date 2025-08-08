// SPDX-License-Identifier: LGPL-3.0-or-later

package atlas

import (
	"context"

	"github.com/czerwonk/atlas_exporter/exporter"
)

// Strategy defines an strategy to retrieve data for generating metrics
type Strategy interface {
	// MeasurementResults gets results for a list of measurements
	MeasurementResults(ctx context.Context, ids []string) ([]*exporter.Measurement, error)
	// IsHealthy returns true if the strategy is healthy and ready to serve metrics
	IsHealthy() bool
}
