# atlas_exporter (fork)

Metric exporter for RIPE Atlas measurement results.

This repository is a fork of the original project with features I need in production including a modernized configuration system. You should probaly not use it, it's very specific to my usage.

Links:
- Original project: https://github.com/czerwonk/atlas_exporter
- This fork (releases): https://github.com/rjocoleman/atlas_exporter/releases
- Container image: `ghcr.io/rjocoleman/atlas_exporter:latest`
- RIPE Labs article on using atlas_exporter: https://labs.ripe.net/Members/daniel_czerwonk/using-ripe-atlas-measurement-results-in-prometheus-with-atlas_exporter

## Changes From Upstream

#### Production Fixes
- **Fixed nil channel deadlock** ([#53](https://github.com/czerwonk/atlas_exporter/issues/53)): Properly initialize reset channel in streaming strategy worker
- **Fixed hot loop from closed channels**: Detect channel closure correctly to prevent 100% CPU usage
- **Removed aggressive reset mechanism**: Data no longer lost on reconnection, measurements persist properly
- **Removed timeout-based reconnections**: Rely on WebSocket's built-in ping/pong heartbeat instead of forcing disconnections

#### Features
- **Kubernetes health checks**: `/healthz` (liveness) and `/readyz` (readiness) endpoints for container orchestration
- **Observability metrics**: Self-monitoring with `atlas_exporter_stream_connected` and `atlas_exporter_last_data_timestamp` metrics
- **Panic recovery**: Resilient to individual measurement failures - one bad measurement won't crash the exporter
- **Graceful shutdown**: Handles SIGTERM/SIGINT for clean shutdown in container environments
- **Stale probe result filtering** ([#59](https://github.com/czerwonk/atlas_exporter/issues/59)): Added `max_result_age` config to filter out old results from non-participating probes
- **NSID support (toggleable)**: Adds NSID (Name Server Identifier) label for DNS measurements. Enabled by default; disable via `dns.nsid_enabled: false` (or `--dns.nsid_enabled=false` / `ATLAS_DNS__NSID_ENABLED=false`).
- **Enhanced debug logging**: Comprehensive logging for troubleshooting measurement retrieval issues

#### Build & Release Improvements
- **Fixed version injection**: Version now properly set via ldflags during build
- **Simplified Docker builds**: Consolidated release workflow using GoReleaser for both binaries and container images
- **Cleaner configuration**: Removed unnecessary wrapper scripts and simplified deployment

**⚠️ High Cardinality Warning**: NSID can add high cardinality to DNS metrics. It is enabled by default; disable via `dns.nsid_enabled: false` (or `--dns.nsid_enabled=false` / `ATLAS_DNS__NSID_ENABLED=false`). The label is only added when present in DNS responses.

## Breaking Changes (This Fork)
- Unified configuration via koanf + pflag; precedence: defaults < YAML < env < flags
- Flag names changed to dotted keys (no backwards compatibility):
  - Examples: `--web.listen_address`, `--web.telemetry_path`, `--streaming.enabled`, `--worker.count`, `--cache.ttl`
- Durations are strings (e.g., `30s`, `10m`, `1h`), not integers (seconds)
- Environment variables use a strict double-underscore segment separator:
  - `ATLAS_WEB__LISTEN_ADDRESS`, `ATLAS_STREAMING__ENABLED`, `ATLAS_METRICS__GO_ENABLED`, `ATLAS_TLS__CERT_FILE`
  - Arrays (eg. measurements): `ATLAS_MEASUREMENTS__0__ID`, `ATLAS_MEASUREMENTS__1__ID`
- YAML schema updated to match dotted keys with underscores inside segment names:
  - See `config.yaml.example` for the canonical schema

These changes intentionally break compatibility with prior config/flag/env names to simplify and standardize configuration.

## Streaming API
Since version 0.8 atlas_exporter also supports retrieving measurement results by RIPE Atlas Streaming API (https://atlas.ripe.net/docs/result-streaming/). Using this feature requires config file mode. All configured measurements are subscribed on start so the latest result for each probe is updated continuously and scrape time is reduced significantly. When a socket.io connection fails a reconnect is initiated. Streaming API is the default for config file mode, it can be disabled by setting `--streaming.enabled=false`.

## Histograms
Since version 1.0 atlas_exporter provides you with histograms of round trip times of the following measurement types:
* DNS
* Ping
* Traceroute
* HTTP

The buckets can be configured in the config file (see below).

Since this feature relies strongly on getting each update for a measurement, the Stream API mode has to be used.
Histogram metrics enables you to calculate percentiles for a specifiv indicator (in our case round trip time). This allows better monitoring of defined service level objectives (e.g. Ping RTT of a specific measurement should be under a specific threshold based on 90% of the requests disregarding the highest 10% -> p90).

For more information:
https://prometheus.io/docs/practices/histograms/

## Install
Download a binary from GitHub Releases: https://github.com/rjocoleman/atlas_exporter/releases

## Docker
To start the server:
```
docker run -d --restart unless-stopped -p 9400:9400 ghcr.io/rjocoleman/atlas_exporter:latest
```
To run in config file mode:
```
docker run -d --name atlas_exporter \
  -e ATLAS_CONFIG_FILE=/etc/atlas/config.yaml \
  -v /path/to/config.yaml:/etc/atlas/config.yaml:ro \
  --restart unless-stopped \
  -p 9400:9400 \
  ghcr.io/rjocoleman/atlas_exporter:latest
```

## Usage
### Start server
```
./atlas_exporter
```
or using config file mode:
```
./atlas_exporter --config.file config.yml
```

### Config file
See `config.yaml.example` for a complete example with all available options.

Basic example for monitoring measurement 8772164:
```YAML
measurements:
  - id: 8772164
histogram_buckets:
  ping:
    rtt:
      - 5.0
      - 10.0
      - 25.0
      - 50.0
      - 100.0
filter_invalid_results: true
max_result_age: 10m
```

### Call metrics URI
when using config file mode:
```
curl http://127.0.0.1:9400/metrics
```
or ad hoc for measuremnt 8772164:
```
curl http://127.0.0.1:9400/metrics?measurement_id=8772164
```
in both cases the result should look similar to this one:
```
# HELP atlas_traceroute_hops Number of hops
# TYPE atlas_traceroute_hops gauge
atlas_traceroute_hops{asn="1101",dst_addr="8.8.8.8",dst_name="8.8.8.8",ip_version="4",measurement="8772164",probe="6031"} 9
atlas_traceroute_hops{asn="11051",dst_addr="8.8.8.8",dst_name="8.8.8.8",ip_version="4",measurement="8772164",probe="17833"} 8
atlas_traceroute_hops{asn="111",dst_addr="8.8.8.8",dst_name="8.8.8.8",ip_version="4",measurement="8772164",probe="6231"} 9
atlas_traceroute_hops{asn="11427",dst_addr="8.8.8.8",dst_name="8.8.8.8",ip_version="4",measurement="8772164",probe="1121"} 13
atlas_traceroute_hops{asn="12337",dst_addr="8.8.8.8",dst_name="8.8.8.8",ip_version="4",measurement="8772164",probe="267"} 13
atlas_traceroute_hops{asn="1257",dst_addr="8.8.8.8",dst_name="8.8.8.8",ip_version="4",measurement="8772164",probe="140"} 11
atlas_traceroute_hops{asn="12586",dst_addr="8.8.8.8",dst_name="8.8.8.8",ip_version="4",measurement="8772164",probe="2088"} 13
atlas_traceroute_hops{asn="12597",dst_addr="8.8.8.8",dst_name="8.8.8.8",ip_version="4",measurement="8772164",probe="2619"} 10
atlas_traceroute_hops{asn="12714",dst_addr="8.8.8.8",dst_name="8.8.8.8",ip_version="4",measurement="8772164",probe="2684"} 9
atlas_traceroute_hops{asn="133752",dst_addr="8.8.8.8",dst_name="8.8.8.8",ip_version="4",measurement="8772164",probe="6191"} 14

...
```

## Features
* ping measurements (success, min/max/avg latency, dups, size)
* traceroute measurements (success, hop count, rtt)
* ntp (delay, derivation, ntp version)
* dns (success, rtt, nsid [optional] - Name Server Identifier from EDNS0, displayed as ASCII if printable or hex otherwise; toggle via `dns.nsid_enabled`)
* http (return code, rtt, http version, header size, body size)
* sslcert (alert, rtt)

### Exporter Observability Metrics

The exporter provides its own operational metrics:

* `atlas_exporter_stream_connected{measurement_id="X"}` - Gauge showing if websocket is connected (1) or not (0)
* `atlas_exporter_last_data_timestamp{measurement_id="X"}` - Gauge with Unix timestamp of last received data

These metrics help monitor the exporter's health and can be used for alerting on connection issues or stale data.

## Health Checks

The exporter provides Kubernetes-compatible health endpoints:

* `/healthz` - Liveness probe (always returns 200 if the server is running)
* `/readyz` - Readiness probe (returns 200 when websocket connections are established in streaming mode)

Optional data freshness checking can be configured:
```yaml
# In config.yaml
health:
  max_data_age: 30m  # Fail readiness if no data received for 30 minutes
```
Or via CLI flag: `--health.max_data_age=30m`

### Configuration Sources and Precedence
- Defaults < YAML file (`--config.file` or `ATLAS_CONFIG_FILE`) < Environment (`ATLAS_*`) < Flags
- Environment mapping (strict): use `__` between path segments; single `_` stays in the name
  - `ATLAS_WEB__LISTEN_ADDRESS` → `web.listen_address`
  - `ATLAS_STREAMING__ENABLED` → `streaming.enabled`
  - `ATLAS_METRICS__GO_ENABLED` → `metrics.go_enabled`
  - `ATLAS_TLS__CERT_FILE` → `tls.cert_file`
  - Arrays: `ATLAS_MEASUREMENTS__0__ID`, `ATLAS_MEASUREMENTS__1__ID`

## Prometheus configuration

### Ad-Hoc Mode
```yaml
  - job_name: 'atlas_exporter'
    scrape_interval: 5m
    static_configs:
      - targets:
        - 7924888
        - 7924886
    relabel_configs:
      - source_labels: [__address__]
        regex: (.*)(:80)?
        target_label: __param_measurement_id
        replacement: ${1}
      - source_labels: [__param_measurement_id]
        regex: (.*)
        target_label: instance
        replacement: ${1}
      - source_labels: []
        regex: .*
        target_label: __address__
        replacement: atlas-exporter.mytld:9400

```

### Config Mode
```yaml
  - job_name: 'atlas_exporter'
    scrape_interval: 5m
    static_configs:
      - targets:
          - atlas-exporter.mytld:9400
```

## Third Party Components
- Go bindings for RIPE Atlas API: https://github.com/DNS-OARC/ripeatlas
- Prometheus Go client library: https://github.com/prometheus/client_golang

## License
Upstream copyright (c) Daniel Czerwonk. Licensed under [LGPL-3.0-or-later](LICENSE).

## Prometheus
https://prometheus.io/

## The RIPE Atlas Project
https://atlas.ripe.net

## Further Reading
- RIPE Labs: Using RIPE Atlas measurement results in Prometheus with atlas_exporter
  https://labs.ripe.net/Members/daniel_czerwonk/using-ripe-atlas-measurement-results-in-prometheus-with-atlas_exporter
### Kubernetes Shutdown Guidance
- Use `terminationGracePeriodSeconds: 30-60` to allow graceful shutdown. The exporter cancels workers via SIGTERM and performs `http.Server.Shutdown` with a 30s timeout.
- Prefer readiness/liveness probes:
  - Liveness: `/healthz`
  - Readiness: `/readyz` (in streaming mode also considers recent data if `health.max_data_age` > 0)
- Example probes:
  - `readinessProbe: { httpGet: { path: /readyz, port: 9400 }, initialDelaySeconds: 5, periodSeconds: 10 }`
  - `livenessProbe:  { httpGet: { path: /healthz, port: 9400 }, initialDelaySeconds: 5, periodSeconds: 10 }`
