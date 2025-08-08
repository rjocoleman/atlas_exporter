// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"fmt"
	"io"
	"time"

	yaml "gopkg.in/yaml.v2"
)

// Config represents the configuration for the exporter
type Config struct {
	// Measurements is the ids of measurements used as source for metrics generation
	Measurements         []Measurement    `yaml:"measurements"`
	HistogramBuckets     HistogramBuckets `yaml:"histogram_buckets"`
	FilterInvalidResults bool             `yaml:"filter_invalid_results"`
	MaxResultAge         time.Duration    `yaml:"max_result_age,omitempty"`
	HealthMaxDataAge     time.Duration    `yaml:"health_max_data_age,omitempty"`
}

// HistogramBuckets defines buckets for several histograms
type HistogramBuckets struct {
	DNS        RttHistogramBucket `yaml:"dns,omitempty"`
	HTTP       RttHistogramBucket `yaml:"http,omitempty"`
	Ping       RttHistogramBucket `yaml:"ping,omitempty"`
	Traceroute RttHistogramBucket `yaml:"traceroute,omitempty"`
}

// RttHistogramBucket defines buckets for RTT histograms
type RttHistogramBucket struct {
	Rtt []float64 `yaml:"rtt"`
}

// Measurement represents config options for one measurement
type Measurement struct {
	ID string `yaml:"id"`
}

// MeasurementIDs represents all IDs of configured measurements
func (c *Config) MeasurementIDs() []string {
	ids := make([]string, len(c.Measurements))
	for i, m := range c.Measurements {
		ids[i] = m.ID
	}

	return ids
}

// Load loads a config from a reader
func Load(r io.Reader) (*Config, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("could not load config: %v", err)
	}

	c := &Config{
		FilterInvalidResults: true,
	}
	if len(b) == 0 {
		return c, nil
	}

	err = yaml.Unmarshal(b, c)
	if err != nil {
		return nil, fmt.Errorf("could not parse config: %v", err)
	}

	return c, err
}
