package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func newFlagSet() *pflag.FlagSet {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	RegisterFlags(fs)
	return fs
}

func TestDefaultsOnly(t *testing.T) {
	t.Setenv("ATLAS_CONFIG_FILE", "")
	fs := newFlagSet()
	require.NoError(t, fs.Parse([]string{}))

	cfg, err := Load(fs)
	require.NoError(t, err)

	require.Equal(t, ":9400", cfg.Web.ListenAddress)
	require.Equal(t, "/metrics", cfg.Web.TelemetryPath)
	require.Equal(t, uint(8), cfg.Worker.Count)
	require.Equal(t, true, cfg.Streaming.Enabled)
	require.Equal(t, true, cfg.Metrics.GoEnabled)
	require.True(t, cfg.FilterInvalidResults)
}

func TestFileEnvFlagPrecedence(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "cfg.yaml")
	err := os.WriteFile(yamlPath, []byte("web:\n  telemetry_path: /m\nmetrics:\n  go_enabled: false\nlog:\n  level: warn\n"), 0o600)
	require.NoError(t, err)

	// File: go_enabled=false, log.level=warn, telemetry_path=/m
	// Env overrides: go_enabled=true
	// Flag overrides: go_enabled=false, log.level=debug
	t.Setenv("ATLAS_CONFIG_FILE", yamlPath)
	t.Setenv("ATLAS_METRICS__GO_ENABLED", "true")

	fs := newFlagSet()
	require.NoError(t, fs.Parse([]string{"--metrics.go_enabled=false", "--log.level=debug"}))

	cfg, err := Load(fs)
	require.NoError(t, err)

	// precedence: file < env < flags
	require.Equal(t, "/m", cfg.Web.TelemetryPath)
	require.Equal(t, false, cfg.Metrics.GoEnabled) // flag wins over env
	require.Equal(t, "debug", cfg.Log.Level)       // flag overrides file
}

func TestEnvMappingAndParsing(t *testing.T) {
	fs := newFlagSet()
	t.Setenv("ATLAS_WEB__LISTEN_ADDRESS", "127.0.0.1:9999")
	t.Setenv("ATLAS_STREAMING__ENABLED", "false")
	t.Setenv("ATLAS_WORKER__COUNT", "16")
	t.Setenv("ATLAS_CACHE__TTL", "15m")
	require.NoError(t, fs.Parse([]string{}))

	cfg, err := Load(fs)
	require.NoError(t, err)

	require.Equal(t, "127.0.0.1:9999", cfg.Web.ListenAddress)
	require.Equal(t, false, cfg.Streaming.Enabled)
	require.Equal(t, uint(16), cfg.Worker.Count)
	require.Equal(t, int64(15*60), int64(cfg.Cache.TTL.Seconds()))
}

func TestValidation(t *testing.T) {
	fs := newFlagSet()
	require.NoError(t, fs.Parse([]string{"--web.telemetry_path=metrics"}))
	_, err := Load(fs)
	require.Error(t, err)
}

func TestDurationParsing_AllFields(t *testing.T) {
	fs := newFlagSet()
	t.Setenv("ATLAS_CACHE__TTL", "10m")
	t.Setenv("ATLAS_CACHE__CLEANUP", "5m")
	t.Setenv("ATLAS_TIMEOUT", "30s")
	t.Setenv("ATLAS_HEALTH__MAX_DATA_AGE", "1h")
	t.Setenv("ATLAS_MAX_RESULT_AGE", "45m")
	require.NoError(t, fs.Parse([]string{}))

	cfg, err := Load(fs)
	require.NoError(t, err)

	require.Equal(t, int64(600), int64(cfg.Cache.TTL.Seconds()))
	require.Equal(t, int64(300), int64(cfg.Cache.Cleanup.Seconds()))
	require.Equal(t, int64(30), int64(cfg.Timeout.Seconds()))
	require.Equal(t, int64(3600), int64(cfg.Health.MaxDataAge.Seconds()))
	require.Equal(t, int64(2700), int64(cfg.MaxResultAge.Seconds()))
}

func TestBooleanAndNumericParsing(t *testing.T) {
	fs := newFlagSet()
	t.Setenv("ATLAS_STREAMING__ENABLED", "false")
	t.Setenv("ATLAS_METRICS__PROCESS_ENABLED", "false")
	t.Setenv("ATLAS_PROFILING__ENABLED", "true")
	t.Setenv("ATLAS_WORKER__COUNT", "12")
	t.Setenv("ATLAS_STREAMING__BUFFER_SIZE", "200")
	require.NoError(t, fs.Parse([]string{}))

	cfg, err := Load(fs)
	require.NoError(t, err)

	require.False(t, cfg.Streaming.Enabled)
	require.False(t, cfg.Metrics.ProcessEnabled)
	require.True(t, cfg.Profiling.Enabled)
	require.Equal(t, uint(12), cfg.Worker.Count)
	require.Equal(t, uint(200), cfg.Streaming.BufferSize)
}

func TestDefaultsApplied(t *testing.T) {
	fs := newFlagSet()
	require.NoError(t, fs.Parse([]string{}))
	cfg, err := Load(fs)
	require.NoError(t, err)

	require.Equal(t, ":9400", cfg.Web.ListenAddress)
	require.Equal(t, "/metrics", cfg.Web.TelemetryPath)
	require.Equal(t, int64(3600), int64(cfg.Cache.TTL.Seconds()))
	require.Equal(t, int64(300), int64(cfg.Cache.Cleanup.Seconds()))
	require.Equal(t, int64(60), int64(cfg.Timeout.Seconds()))
	require.Equal(t, uint(8), cfg.Worker.Count)
	require.True(t, cfg.Streaming.Enabled)
	require.Equal(t, uint(100), cfg.Streaming.BufferSize)
	require.False(t, cfg.Profiling.Enabled)
	require.True(t, cfg.Metrics.GoEnabled)
	require.True(t, cfg.Metrics.ProcessEnabled)
	require.Equal(t, "info", cfg.Log.Level)
	require.False(t, cfg.TLS.Enabled)
	require.Equal(t, "", cfg.TLS.CertFile)
	require.Equal(t, "", cfg.TLS.KeyFile)
	require.Equal(t, int64(0), int64(cfg.Health.MaxDataAge.Seconds()))
	require.True(t, cfg.FilterInvalidResults)
	require.Equal(t, int64(0), int64(cfg.MaxResultAge.Seconds()))
}

func TestMeasurementIDsHelper(t *testing.T) {
	// empty
	cfg := &Config{}
	require.Len(t, cfg.MeasurementIDs(), 0)

	// single
	cfg = &Config{Measurements: []Measurement{{ID: "123"}}}
	require.Equal(t, []string{"123"}, cfg.MeasurementIDs())

	// multiple
	cfg = &Config{Measurements: []Measurement{{ID: "a"}, {ID: "b"}, {ID: "c"}}}
	require.Equal(t, []string{"a", "b", "c"}, cfg.MeasurementIDs())
}

func TestMeasurementsParsing_YAML(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "cfg.yaml")
	err := os.WriteFile(yamlPath, []byte("measurements:\n  - id: 123\n  - id: 456\n"), 0o600)
	require.NoError(t, err)

	fs := newFlagSet()
	require.NoError(t, fs.Parse([]string{"--config.file=" + yamlPath}))

	cfg, err := Load(fs)
	require.NoError(t, err)
	require.Equal(t, []string{"123", "456"}, cfg.MeasurementIDs())
}

func TestMeasurementsParsing_Env(t *testing.T) {
	fs := newFlagSet()
	t.Setenv("ATLAS_MEASUREMENTS__0__ID", "111")
	t.Setenv("ATLAS_MEASUREMENTS__1__ID", "222")
	require.NoError(t, fs.Parse([]string{}))

	cfg, err := Load(fs)
	require.NoError(t, err)
	require.Equal(t, []string{"111", "222"}, cfg.MeasurementIDs())
}

func TestHistogramBucketsParsing_YAML(t *testing.T) {
	dir := t.TempDir()
	yaml := `histogram_buckets:
  dns:
    rtt: [1.0, 2.0]
  http:
    rtt: [3.0, 4.0]
  ping:
    rtt: [5.0, 6.0]
  traceroute:
    rtt: [7.0, 8.0]
`
	yamlPath := filepath.Join(dir, "cfg.yaml")
	require.NoError(t, os.WriteFile(yamlPath, []byte(yaml), 0o600))

	fs := newFlagSet()
	require.NoError(t, fs.Parse([]string{"--config.file=" + yamlPath}))
	cfg, err := Load(fs)
	require.NoError(t, err)

	require.Equal(t, []float64{1, 2}, cfg.HistogramBuckets.DNS.Rtt)
	require.Equal(t, []float64{3, 4}, cfg.HistogramBuckets.HTTP.Rtt)
	require.Equal(t, []float64{5, 6}, cfg.HistogramBuckets.Ping.Rtt)
	require.Equal(t, []float64{7, 8}, cfg.HistogramBuckets.Traceroute.Rtt)
}

func TestValidation_TLS(t *testing.T) {
	// enabled but missing cert/key
	fs := newFlagSet()
	require.NoError(t, fs.Parse([]string{"--tls.enabled=true"}))
	_, err := Load(fs)
	require.Error(t, err)

	// only cert provided
	dir := t.TempDir()
	cert := filepath.Join(dir, "cert.pem")
	key := filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(cert, []byte("dummy"), 0o600))

	fs = newFlagSet()
	require.NoError(t, fs.Parse([]string{"--tls.enabled=true", "--tls.cert_file=" + cert}))
	_, err = Load(fs)
	require.Error(t, err)

	// both provided
	require.NoError(t, os.WriteFile(key, []byte("dummy"), 0o600))
	fs = newFlagSet()
	require.NoError(t, fs.Parse([]string{"--tls.enabled=true", "--tls.cert_file=" + cert, "--tls.key_file=" + key}))
	_, err = Load(fs)
	require.NoError(t, err)
}

func TestValidation_NegativeDuration(t *testing.T) {
	fs := newFlagSet()
	t.Setenv("ATLAS_TIMEOUT", "-1s")
	require.NoError(t, fs.Parse([]string{}))
	_, err := Load(fs)
	require.Error(t, err)
}

func TestValidation_HistogramBuckets_Invalid(t *testing.T) {
	dir := t.TempDir()
	yaml := `histogram_buckets:\n  dns:\n    rtt: [1.0, -2.0]\n`
	path := filepath.Join(dir, "bad.yaml")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	fs := newFlagSet()
	require.NoError(t, fs.Parse([]string{"--config.file=" + path}))
	_, err := Load(fs)
	require.Error(t, err)

	// non-decreasing
	yaml = `histogram_buckets:\n  ping:\n    rtt: [2.0, 1.0]\n`
	path2 := filepath.Join(dir, "bad2.yaml")
	require.NoError(t, os.WriteFile(path2, []byte(yaml), 0o600))
	fs = newFlagSet()
	require.NoError(t, fs.Parse([]string{"--config.file=" + path2}))
	_, err = Load(fs)
	require.Error(t, err)
}

func TestMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	// Definitely invalid YAML
	require.NoError(t, os.WriteFile(path, []byte(":\n-"), 0o600))
	fs := newFlagSet()
	require.NoError(t, fs.Parse([]string{"--config.file=" + path}))
	_, err := Load(fs)
	require.Error(t, err)
}

func TestNonExistentConfigFile(t *testing.T) {
	fs := newFlagSet()
	require.NoError(t, fs.Parse([]string{"--config.file=/does/not/exist.yaml"}))
	_, err := Load(fs)
	require.Error(t, err)
}

func TestInvalidEnvTypes(t *testing.T) {
	fs := newFlagSet()
	t.Setenv("ATLAS_WORKER__COUNT", "abc")
	require.NoError(t, fs.Parse([]string{}))
	_, err := Load(fs)
	require.Error(t, err)
}

func TestInvalidDurationFormat(t *testing.T) {
	fs := newFlagSet()
	t.Setenv("ATLAS_CACHE__TTL", "not-a-duration")
	require.NoError(t, fs.Parse([]string{}))
	_, err := Load(fs)
	require.Error(t, err)
}
