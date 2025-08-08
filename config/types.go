// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import "time"

// Config represents the full configuration for the exporter
type Config struct {
	Web struct {
		ListenAddress string `koanf:"listen_address" yaml:"listen_address"`
		TelemetryPath string `koanf:"telemetry_path" yaml:"telemetry_path"`
	} `koanf:"web" yaml:"web"`

	Cache struct {
		TTL     time.Duration `koanf:"ttl" yaml:"ttl"`
		Cleanup time.Duration `koanf:"cleanup" yaml:"cleanup"`
	} `koanf:"cache" yaml:"cache"`

	Timeout time.Duration `koanf:"timeout" yaml:"timeout"`

	Worker struct {
		Count uint `koanf:"count" yaml:"count"`
	} `koanf:"worker" yaml:"worker"`

	Streaming struct {
		Enabled    bool `koanf:"enabled" yaml:"enabled"`
		BufferSize uint `koanf:"buffer_size" yaml:"buffer_size"`
	} `koanf:"streaming" yaml:"streaming"`

	Profiling struct {
		Enabled bool `koanf:"enabled" yaml:"enabled"`
	} `koanf:"profiling" yaml:"profiling"`

	Metrics struct {
		GoEnabled      bool `koanf:"go_enabled" yaml:"go_enabled"`
		ProcessEnabled bool `koanf:"process_enabled" yaml:"process_enabled"`
	} `koanf:"metrics" yaml:"metrics"`

	Log struct {
		Level string `koanf:"level" yaml:"level"`
	} `koanf:"log" yaml:"log"`

	TLS struct {
		Enabled  bool   `koanf:"enabled" yaml:"enabled"`
		CertFile string `koanf:"cert_file" yaml:"cert_file"`
		KeyFile  string `koanf:"key_file" yaml:"key_file"`
	} `koanf:"tls" yaml:"tls"`

	Health struct {
		MaxDataAge time.Duration `koanf:"max_data_age" yaml:"max_data_age"`
	} `koanf:"health" yaml:"health"`

	// Existing config fields preserved and names aligned to new schema
	HistogramBuckets HistogramBuckets `koanf:"histogram_buckets" yaml:"histogram_buckets"`
	Measurements     []Measurement    `koanf:"measurements" yaml:"measurements"`

	// Behavior flags
	FilterInvalidResults bool          `koanf:"filter_invalid_results" yaml:"filter_invalid_results"`
	MaxResultAge         time.Duration `koanf:"max_result_age" yaml:"max_result_age"`
}

// HistogramBuckets defines buckets for several histograms
type HistogramBuckets struct {
	DNS        RttHistogramBucket `yaml:"dns,omitempty" koanf:"dns,omitempty"`
	HTTP       RttHistogramBucket `yaml:"http,omitempty" koanf:"http,omitempty"`
	Ping       RttHistogramBucket `yaml:"ping,omitempty" koanf:"ping,omitempty"`
	Traceroute RttHistogramBucket `yaml:"traceroute,omitempty" koanf:"traceroute,omitempty"`
}

// RttHistogramBucket defines buckets for RTT histograms
type RttHistogramBucket struct {
	Rtt []float64 `yaml:"rtt" koanf:"rtt"`
}

// Measurement represents config options for one measurement
type Measurement struct {
	ID string `yaml:"id" koanf:"id"`
}

// MeasurementIDs represents all IDs of configured measurements
func (c *Config) MeasurementIDs() []string {
	ids := make([]string, len(c.Measurements))
	for i, m := range c.Measurements {
		ids[i] = m.ID
	}
	return ids
}
