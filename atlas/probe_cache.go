// SPDX-License-Identifier: LGPL-3.0-or-later

package atlas

import (
	"time"

	"github.com/czerwonk/atlas_exporter/probe"
	log "github.com/sirupsen/logrus"
)

var cache *probe.Cache

// InitCache initializes the cache
func InitCache(ttl, cleanup time.Duration) {
	cache = probe.NewCache(ttl)
	startCacheCleanupFunc(cleanup)
}

func startCacheCleanupFunc(d time.Duration) {
	go func() {
		ticker := time.NewTicker(d)
		defer ticker.Stop()

		for range ticker.C {
			log.Infoln("Cleaning up cache...")
			r := cache.CleanUp()
			log.Infof("Items removed: %d", r)
		}
	}()
}
