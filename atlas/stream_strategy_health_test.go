package atlas

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/czerwonk/atlas_exporter/config"
)

func TestStreamingIsHealthy_ConnectedNoAge(t *testing.T) {
	s := &streamingStrategy{cfg: &config.Config{}}
	atomic.StoreInt32(&s.connectedWorkers, 1)
	if !s.IsHealthy() {
		t.Fatalf("expected healthy when connected and no age constraint")
	}
}

func TestStreamingIsHealthy_DataAge(t *testing.T) {
	s := &streamingStrategy{cfg: &config.Config{}}
	s.cfg.Health.MaxDataAge = 10 * time.Second
	atomic.StoreInt32(&s.connectedWorkers, 1)

	// No data yet -> not healthy
	if s.IsHealthy() {
		t.Fatalf("expected not healthy without data timestamp")
	}

	// Recent data -> healthy
	now := time.Now().Unix()
	atomic.StoreInt64(&s.lastDataTime, now)
	if !s.IsHealthy() {
		t.Fatalf("expected healthy with fresh data")
	}

	// Stale data -> not healthy
	atomic.StoreInt64(&s.lastDataTime, now-int64((s.cfg.Health.MaxDataAge+time.Second).Seconds()))
	if s.IsHealthy() {
		t.Fatalf("expected not healthy with stale data")
	}
}
