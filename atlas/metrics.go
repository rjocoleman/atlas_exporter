// SPDX-License-Identifier: LGPL-3.0-or-later

package atlas

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// StreamConnectedGauge tracks websocket connection status per measurement
	StreamConnectedGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "atlas_exporter_stream_connected",
			Help: "Whether the websocket stream is connected (1) or not (0) for a measurement",
		},
		[]string{"measurement_id"},
	)

	// LastDataTimestampGauge tracks when data was last received per measurement
	LastDataTimestampGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "atlas_exporter_last_data_timestamp",
			Help: "Unix timestamp of when data was last received for a measurement",
		},
		[]string{"measurement_id"},
	)
)
