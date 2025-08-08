// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	mapstructure "github.com/go-viper/mapstructure/v2"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	envv2 "github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
)

const (
	envPrefix = "ATLAS_"
)

// RegisterFlags defines all supported flags with default values.
// These names are the canonical dotted keys.
func RegisterFlags(fs *pflag.FlagSet) {
	d := Defaults()

	fs.String("web.listen_address", d["web.listen_address"].(string), "Address to expose metrics and web interface.")
	fs.String("web.telemetry_path", d["web.telemetry_path"].(string), "Path under which to expose metrics.")
	fs.String("cache.ttl", d["cache.ttl"].(string), "Cache TTL (duration e.g. 3600s)")
	fs.String("cache.cleanup", d["cache.cleanup"].(string), "Cache cleanup interval (duration)")
	fs.String("timeout", d["timeout"].(string), "Timeout for metrics requests (duration)")
	fs.Uint("worker.count", uint(d["worker.count"].(int)), "Number of goroutines retrieving probe information")
	fs.Bool("streaming.enabled", d["streaming.enabled"].(bool), "Retrieve data via Atlas Streaming API")
	fs.Uint("streaming.buffer_size", uint(d["streaming.buffer_size"].(int)), "Buffer size for streaming worker channel")
	fs.Bool("profiling.enabled", d["profiling.enabled"].(bool), "Enable pprof endpoints")
	fs.Bool("metrics.go_enabled", d["metrics.go_enabled"].(bool), "Enable Go runtime metrics")
	fs.Bool("metrics.process_enabled", d["metrics.process_enabled"].(bool), "Enable process metrics")
	fs.String("log.level", d["log.level"].(string), "Log level: debug|info|warn|error|fatal")
	fs.Bool("tls.enabled", d["tls.enabled"].(bool), "Enable TLS for HTTP server")
	fs.String("tls.cert_file", d["tls.cert_file"].(string), "Path to TLS certificate file")
	fs.String("tls.key_file", d["tls.key_file"].(string), "Path to TLS key file")
	fs.String("health.max_data_age", d["health.max_data_age"].(string), "Max data age for readiness check (duration, 0s=disabled)")
	fs.Bool("filter_invalid_results", d["filter_invalid_results"].(bool), "Filter invalid results by IP version capability")
	fs.String("max_result_age", d["max_result_age"].(string), "Skip results older than this (duration, 0s=disabled)")
	fs.Bool("dns.nsid_enabled", d["dns.nsid_enabled"].(bool), "Enable DNS NSID label (may increase cardinality)")

	// Special: config file path, not part of Config struct (used for file provider)
	fs.String("config.file", "", "Path to YAML config file")
}

// Load loads configuration from defaults, optional file, env and flags with clear precedence.
func Load(fs *pflag.FlagSet) (*Config, error) {
	k := koanf.New(".")

	// 1. defaults
	if err := k.Load(confmap.Provider(Defaults(), "."), nil); err != nil {
		return nil, fmt.Errorf("load defaults: %w", err)
	}

	// Resolve config file path from flag or env
	filePath := ""
	if f := fs.Lookup("config.file"); f != nil {
		filePath = f.Value.String()
	}
	if filePath == "" {
		filePath = os.Getenv(envPrefix + "CONFIG_FILE")
	}
	if filePath != "" {
		// Expand and load if exists
		if abs, err := filepath.Abs(filePath); err == nil {
			filePath = abs
		}
		if err := k.Load(file.Provider(filePath), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("load yaml file %s: %w", filePath, err)
		}
	}

	// 3. env (use env/v2 provider with prefix + transformer)
	envProvider := envv2.Provider(".", envv2.Opt{
		Prefix:        envPrefix,
		TransformFunc: envTransform,
	})
	if err := k.Load(envProvider, nil); err != nil {
		return nil, fmt.Errorf("load env: %w", err)
	}

	// 4. flags (expect fs.Parse() already done in main)
	if fs != nil {
		if err := k.Load(posflag.Provider(fs, ".", k), nil); err != nil {
			return nil, fmt.Errorf("load flags: %w", err)
		}
	}

	// Normalize structures that env can't express naturally
	normalizeArrays(k)

	// Unmarshal with proper duration hooks
	var cfg Config
	dc := &mapstructure.DecoderConfig{
		TagName:          "koanf",
		WeaklyTypedInput: true,
		DecodeHook:       mapstructure.StringToTimeDurationHookFunc(),
	}
	dc.Result = &cfg
	if err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{DecoderConfig: dc}); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	if err := Validate(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// normalizeArrays converts maps with numeric keys created via env into arrays where needed.
func normalizeArrays(k *koanf.Koanf) {
	// measurements: expect array of objects
	if v := k.Get("measurements"); v != nil {
		if m, ok := v.(map[string]any); ok {
			// Convert map of index->object into a slice ordered by index
			type kv struct {
				idx int
				val any
			}
			arr := make([]kv, 0, len(m))
			for ks, vv := range m {
				// try parse index
				var idx = -1
				if i, err := strconv.Atoi(ks); err == nil {
					idx = i
				}
				if idx >= 0 {
					arr = append(arr, kv{idx: idx, val: vv})
				}
			}
			if len(arr) > 0 {
				sort.Slice(arr, func(i, j int) bool { return arr[i].idx < arr[j].idx })
				out := make([]any, arr[len(arr)-1].idx+1)
				for _, e := range arr {
					if e.idx >= 0 && e.idx < len(out) {
						out[e.idx] = e.val
					}
				}
				_ = k.Set("measurements", out)
			}
		}
	}
}

// envTransform maps env like ATLAS_WEB__LISTEN_ADDRESS to web.listen_address.
// Convention: use double underscore "__" as the path separator. Single underscores are preserved
// inside segment names (eg: CERT_FILE).
func envTransform(k, v string) (string, any) {
	k = strings.TrimPrefix(strings.ToUpper(k), envPrefix)
	k = strings.ToLower(k)
	segs := strings.Split(k, "__")
	key := strings.Join(segs, ".")
	return key, v
}

// Validate performs light validation on the final config
func Validate(c *Config) error {
	if c.Web.TelemetryPath == "" || !strings.HasPrefix(c.Web.TelemetryPath, "/") {
		return errors.New("web.telemetry_path must start with '/'")
	}
	if c.TLS.Enabled {
		if c.TLS.CertFile == "" || c.TLS.KeyFile == "" {
			return errors.New("tls enabled but cert_file or key_file missing")
		}
	}
	// histogram buckets must be non-negative and non-decreasing
	for name, b := range map[string][]float64{
		"dns.rtt":        c.HistogramBuckets.DNS.Rtt,
		"http.rtt":       c.HistogramBuckets.HTTP.Rtt,
		"ping.rtt":       c.HistogramBuckets.Ping.Rtt,
		"traceroute.rtt": c.HistogramBuckets.Traceroute.Rtt,
	} {
		if !isNonDecreasingNonNegative(b) {
			return fmt.Errorf("histogram_buckets.%s must be non-negative and non-decreasing", name)
		}
	}
	// durations are >= 0 implicitly by type; but ensure not negative due to parsing
	if c.Cache.TTL < 0 || c.Cache.Cleanup < 0 || c.Timeout < 0 || c.MaxResultAge < 0 || c.Health.MaxDataAge < 0 {
		return errors.New("duration values must be >= 0")
	}
	return nil
}

func isNonDecreasingNonNegative(vals []float64) bool {
	prev := -1.0
	for i, v := range vals {
		if v < 0 {
			return false
		}
		if i > 0 && v < prev {
			return false
		}
		prev = v
	}
	return true
}
