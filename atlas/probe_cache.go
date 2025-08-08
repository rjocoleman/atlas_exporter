// SPDX-License-Identifier: LGPL-3.0-or-later

package atlas

import (
	"context"
	"time"

	"github.com/czerwonk/atlas_exporter/probe"
	log "github.com/sirupsen/logrus"
)

var cache *probe.Cache

// InitCache initializes the cache
func InitCache(ctx context.Context, ttl, cleanup time.Duration) {
	cache = probe.NewCache(ttl)
	startCacheCleanupFunc(ctx, cleanup)
}

func startCacheCleanupFunc(ctx context.Context, d time.Duration) {
	go func() {
		ticker := time.NewTicker(d)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				log.Debugln("Cleaning up cache...")
				r := cache.CleanUp()
				if r > 0 {
					log.Infof("Cache items removed: %d", r)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
