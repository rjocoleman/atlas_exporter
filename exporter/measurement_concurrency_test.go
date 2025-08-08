package exporter

import (
    "sync"
    "testing"

    mdms "github.com/DNS-OARC/ripeatlas/measurement"
    "github.com/czerwonk/atlas_exporter/probe"
    "github.com/prometheus/client_golang/prometheus"
)

// dummyExporter is a no-op exporter for tests
type dummyExporter struct{}

func (d dummyExporter) Export(res *mdms.Result, pr *probe.Probe, ch chan<- prometheus.Metric) {}
func (d dummyExporter) Describe(ch chan<- *prometheus.Desc)                          {}

// TestMeasurementConcurrentAddCollect ensures no races between Add and Collect
func TestMeasurementConcurrentAddCollect(t *testing.T) {
    m := NewMeasurement(dummyExporter{})
    p := &probe.Probe{ID: 1}

    // use a zero-value measurement.Result which returns zero IDs/timestamps
    res := &mdms.Result{}

    // run concurrent writers and readers
    var wg sync.WaitGroup
    stop := make(chan struct{})

    // writers
    for i := 0; i < 8; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := 0; j < 1000; j++ {
                m.Add(res, p)
            }
        }()
    }

    // readers
    for i := 0; i < 4; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            ch := make(chan prometheus.Metric, 100)
            for j := 0; j < 1000; j++ {
                m.Collect(ch)
            }
            close(ch)
        }()
    }

    close(stop)
    wg.Wait()
}

