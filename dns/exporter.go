// SPDX-License-Identifier: LGPL-3.0-or-later

package dns

import (
	"encoding/hex"
	"strconv"

	"github.com/DNS-OARC/ripeatlas/measurement"
	"github.com/DNS-OARC/ripeatlas/measurement/dns"
	"github.com/czerwonk/atlas_exporter/probe"
	mdns "github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	labels      []string
	successDesc *prometheus.Desc
	rttDesc     *prometheus.Desc
)

func init() {
	labels = []string{"measurement", "probe", "dst_addr", "asn", "ip_version", "country_code", "lat", "long", "nsid"}

	successDesc = prometheus.NewDesc(prometheus.BuildFQName(ns, sub, "success"), "Destination was reachable", labels, nil)
	rttDesc = prometheus.NewDesc(prometheus.BuildFQName(ns, sub, "rtt"), "Roundtrip time in ms", labels, nil)
}

type dnsExporter struct {
	id string
}

// Export exports a prometheus metric
func (m *dnsExporter) Export(res *measurement.Result, probe *probe.Probe, ch chan<- prometheus.Metric) {
	var nsid string
	if res.DnsResult() != nil {
		nsid = extractNsid(res.DnsResult())
	}

	labelValues := []string{
		m.id,
		strconv.Itoa(probe.ID),
		res.DstAddr(),
		strconv.Itoa(probe.ASNForIPVersion(res.Af())),
		strconv.Itoa(res.Af()),
		probe.CountryCode,
		probe.Latitude(),
		probe.Longitude(),
		nsid,
	}

	var rtt float64
	if res.DnsResult() != nil {
		rtt = res.DnsResult().Rt()
	}

	if rtt > 0 {
		ch <- prometheus.MustNewConstMetric(successDesc, prometheus.GaugeValue, 1, labelValues...)
		ch <- prometheus.MustNewConstMetric(rttDesc, prometheus.GaugeValue, rtt, labelValues...)
	} else {
		ch <- prometheus.MustNewConstMetric(successDesc, prometheus.GaugeValue, 0, labelValues...)
	}
}

// Describe exports metric descriptions for Prometheus
func (m *dnsExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- successDesc
	ch <- rttDesc
}

// extractNsid extracts NSID from DNS result using UnpackAbuf
func extractNsid(result *dns.Result) string {
	msg, err := result.UnpackAbuf()
	if err != nil || msg == nil {
		return ""
	}

	if opt := msg.IsEdns0(); opt != nil {
		for _, o := range opt.Option {
			if e, ok := o.(*mdns.EDNS0_NSID); ok {
				// EDNS0_NSID.Nsid is always hex-encoded
				if raw, err := hex.DecodeString(e.Nsid); err == nil {
					// Return ASCII if printable, else hex
					if isASCIIPrintable(raw) {
						return string(raw)
					}
					return hex.EncodeToString(raw)
				}
			}
		}
	}
	return ""
}

// isASCIIPrintable checks if all bytes are ASCII printable characters (32-126)
func isASCIIPrintable(data []byte) bool {
	for _, b := range data {
		if b < 32 || b > 126 {
			return false
		}
	}
	return true
}
