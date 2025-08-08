// SPDX-License-Identifier: LGPL-3.0-or-later

package config

// Defaults returns the default configuration as a flat map of canonical keys
func Defaults() map[string]any {
	return map[string]any{
		"web.listen_address":      ":9400",
		"web.telemetry_path":      "/metrics",
		"cache.ttl":               "3600s",
		"cache.cleanup":           "300s",
		"timeout":                 "60s",
		"worker.count":            8,
		"streaming.enabled":       true,
		"streaming.buffer_size":   100,
		"profiling.enabled":       false,
		"metrics.go_enabled":      true,
		"metrics.process_enabled": true,
		"log.level":               "info",
		"tls.enabled":             false,
		"tls.cert_file":           "",
		"tls.key_file":            "",
		"health.max_data_age":     "0s",
		"filter_invalid_results":  true,
		"max_result_age":          "0s",
		"dns.nsid_enabled":        true,
		// hist buckets default to empty; measurements default to empty
	}
}
