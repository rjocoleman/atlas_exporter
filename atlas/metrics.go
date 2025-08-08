// SPDX-License-Identifier: LGPL-3.0-or-later

package atlas

import (
	"github.com/prometheus/client_golang/prometheus"
	"runtime"
	"runtime/debug"
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

	// BuildInfoGauge exposes static build information
	BuildInfoGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "atlas_exporter_build_info",
			Help: "Build info. Value is always 1 with labels: version, goversion, vcs_revision",
		},
		[]string{"version", "goversion", "vcs_revision"},
	)

	// ScrapeBuildDuration measures time to fetch measurements and build registry (not HTTP write time)
	ScrapeBuildDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "atlas_exporter_scrape_build_duration_seconds",
			Help:    "Time to retrieve measurements and build metrics registry",
			Buckets: prometheus.DefBuckets,
		},
	)
)

// SetBuildInfo sets the build_info gauge with version, go version and vcs revision
func SetBuildInfo(version string) {
	goVersion := runtime.Version()
	vcs := "unknown"
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			if s.Key == "vcs.revision" && s.Value != "" {
				vcs = s.Value
				break
			}
		}
	}
	BuildInfoGauge.WithLabelValues(version, goVersion, vcs).Set(1)
}
