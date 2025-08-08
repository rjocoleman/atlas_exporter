// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/czerwonk/atlas_exporter/atlas"
	"github.com/czerwonk/atlas_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	_ "net/http/pprof"
)

var version = "dev"

var (
	showVersion = pflag.Bool("version", false, "Print version information.")
	cfg         *config.Config
	strategy    atlas.Strategy
)

func init() {}

func main() {
	// Register canonical flags and parse
	config.RegisterFlags(pflag.CommandLine)
	pflag.Parse()

	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	// Load configuration with precedence: defaults < file < env < flags
	c, err := config.Load(pflag.CommandLine)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	cfg = c

	setLogLevel(cfg.Log.Level)

	log.Debugf("Configured measurements: %v", cfg.MeasurementIDs())

	if cfg.Streaming.Enabled {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		strategy = atlas.NewStreamingStrategy(ctx, cfg, cfg.Streaming.BufferSize)
	} else {
		strategy = atlas.NewRequestStrategy(cfg, cfg.Worker.Count)
	}

	if !cfg.Profiling.Enabled {
		http.DefaultServeMux = http.NewServeMux()
	}

	startServer()
}

func printVersion() {
	fmt.Println("atlas_exporter")
	fmt.Printf("Version: %s\n", version)
	fmt.Println("Author(s): Daniel Czerwonk, Robert Coleman")
	fmt.Println("Metric exporter for RIPE Atlas measurements")
	fmt.Println("This software uses Go bindings from the DNS-OARC project (https://github.com/DNS-OARC/ripeatlas)")
}

// legacy loadConfig removed; using koanf loader in main

func startServer() {
	log.Infof("Starting atlas exporter (Version: %s)", version)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
            <head><title>RIPE Atlas Exporter (Version ` + version + `)</title></head>
            <body>
            <h1>RIPE Atlas Exporter</h1>
            <h2>Example</h2>
            <p>Metrics for measurement configured in configuration file:</p>
            <p><a href="` + cfg.Web.TelemetryPath + `">` + r.Host + cfg.Web.TelemetryPath + `</a></p>
            <p>Metrics for measurement with id 8809582:</p>
            <p><a href="` + cfg.Web.TelemetryPath + `?measurement_id=8809582">` + r.Host + cfg.Web.TelemetryPath + `?measurement_id=8809582</a></p>
            <h2>More information</h2>
            <p><a href="https://github.com/rjocoleman/atlas_exporter">github.com/rjocoleman/atlas_exporter</a></p>
            </body>
            </html>`))
	})
	http.HandleFunc(cfg.Web.TelemetryPath, errorHandler(handleMetricsRequest))

	// Health check endpoints
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if strategy != nil && strategy.IsHealthy() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ready\n"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready\n"))
		}
	})

	log.Infof("Cache TTL: %v", cfg.Cache.TTL)
	log.Infof("Cache cleanup interval: %v", cfg.Cache.Cleanup)
	atlas.InitCache(cfg.Cache.TTL, cfg.Cache.Cleanup)

	log.Infof("Listening for %s on %s (TLS: %v)", cfg.Web.TelemetryPath, cfg.Web.ListenAddress, cfg.TLS.Enabled)
	if cfg.TLS.Enabled {
		log.Fatal(http.ListenAndServeTLS(cfg.Web.ListenAddress, cfg.TLS.CertFile, cfg.TLS.KeyFile, nil))
		return
	}

	log.Fatal(http.ListenAndServe(cfg.Web.ListenAddress, nil))
}

func errorHandler(f func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := f(w, r)

		if err != nil {
			log.Errorln(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func handleMetricsRequest(w http.ResponseWriter, r *http.Request) error {
	id := r.URL.Query().Get("measurement_id")
	log.Debugf("handleMetricsRequest called with measurement_id=%s", id)

	s := strategy

	ids := []string{}
	if len(id) > 0 {
		ids = append(ids, id)
		s = atlas.NewRequestStrategy(cfg, cfg.Worker.Count)
		log.Debugf("Using request strategy for specific measurement: %s", id)
	} else {
		ids = append(ids, cfg.MeasurementIDs()...)
		log.Debugf("Using streaming strategy for configured measurements: %v", ids)
	}

	if len(ids) == 0 {
		log.Debugf("No measurement IDs to query")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	log.Debugf("Requesting measurements for IDs: %v", ids)
	measurements, err := s.MeasurementResults(ctx, ids)
	if err != nil {
		return err
	}
	log.Debugf("Got %d measurements back from strategy", len(measurements))

	reg := prometheus.NewRegistry()

	// add process metrics
	if cfg.Metrics.ProcessEnabled {
		processCollector := collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})
		reg.MustRegister(processCollector)
	}

	// add go collector metrics
	if cfg.Metrics.GoEnabled {
		goCollector := collectors.NewGoCollector()
		reg.MustRegister(goCollector)
	}

	// Add exporter observability metrics
	reg.MustRegister(atlas.StreamConnectedGauge)
	reg.MustRegister(atlas.LastDataTimestampGauge)

	if len(measurements) > 0 {
		c := newCollector(measurements)
		reg.MustRegister(c)
	}

	l := log.New()
	l.Level = log.ErrorLevel

	promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		ErrorLog:      l,
		ErrorHandling: promhttp.ContinueOnError}).ServeHTTP(w, r)

	return nil
}
