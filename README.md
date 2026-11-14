# Optimum Gateway Observability

Lightweight observability for the [Optimum Gateway](https://getoptimum.xyz/) sidecar
(mump2p / RLNC block propagation). This repo offers three ways to monitor a gateway,
from "no custom code" to a small hardened exporter.

> **Scope & honesty:** these tools provide **liveness / stall detection** (is the gateway
> alive, meshed, and receiving blocks?). They do **NOT** measure arrival-latency deltas or
> prove ROI vs gossipsub — `last_block_age_sec` is a freshness gauge, not a latency
> measurement. For ROI, use the Optimum Console per-validator methodology or your CL
> client's own block-arrival timing.

## Which path should I use?

| Path | What it is | When to choose |
|---|---|---|
| **C — Native stack (recommended)** | blackbox_exporter + json_exporter + the gateway's own /metrics | Production. Zero custom Go to audit/maintain. |
| **B — Hardened exporter** | Small Go exporter exposing /health + /self_info as gauges | You want one binary + the bundled dashboard. |
| **A — Demo** | docker-compose.demo.yml (exporter+Prometheus+Grafana) | Learning / local evaluation only. |

**First, check what you already have.** The gateway ships native Prometheus metrics:

    curl -s http://127.0.0.1:48123/metrics | head -50
    curl -s http://127.0.0.1:48123/health | jq

If native /metrics covers your needs, just add a scrape job for :48123 and stop.
Keep the telemetry listener bound to 127.0.0.1 (it is unauthenticated).

## Path C — native stack (recommended)

    cd deploy/native
    docker compose up -d

Grafana auto-provisions the Prometheus datasource and the dashboard (folder
"Optimum") — open http://localhost:3000 and it is already loaded. In production,
scrape from your EXISTING Prometheus and import deploy/native/dashboard.json into
your EXISTING Grafana. Alerts: deploy/native/alerts.yml (probe_success based).

## Path B — hardened exporter

    docker compose up -d          # exporter only, bound to 127.0.0.1:9188

Point GATEWAY_URL at your gateway. Scrape :9188 from your existing Prometheus; import
dashboard.json. Alerts: alerts.yml (uses optimum_gateway_up, which goes to 0 on failure).

Run only the exporter on a validator host — never the demo Prometheus/Grafana.

## Path A — demo (NOT for production)

    docker compose -f docker-compose.demo.yml up -d

Grafana auto-loads the datasource + dashboard (folder "Optimum"); no manual
import. Ships admin/admin and is for local evaluation only.

## Metrics (Path B)

| Metric | Type | Meaning |
|---|---|---|
| optimum_gateway_up | Gauge | 1 if last scrape succeeded, else 0 (trustworthy down-alert) |
| optimum_gateway_last_block_age_sec | Gauge | Seconds since last block (stall detector) |
| optimum_gateway_peers_connected | Gauge | mump2p mesh peers |
| optimum_gateway_health_status | Gauge | 1 healthy / 0 otherwise (cleared on fetch failure) |
| optimum_gateway_uptime_seconds | Gauge | Process uptime (resets => restart) |
| optimum_gateway_api_errors_total | Counter | Failed telemetry fetches |

## Security posture

Read-only GETs to the local gateway; never touches validator keys; no outbound calls
beyond the gateway. Exporter runs as non-root. Bind metrics to loopback / your monitoring
VLAN. Pin image digests and review the code before fleet rollout; canary on one host first.

## Development

    go test ./...
    go vet ./...

CI runs vet + tests + build + docker build on every push/PR.

## Production Grafana (external Prometheus)

    cp .env.example .env   # set PROMETHEUS_URL + a real admin password; keep .env out of git
    docker compose -f docker-compose.grafana.yml up -d

Runs ONLY Grafana against your existing Prometheus (URL via env). Both dashboards
auto-provision into folder "Optimum". Datasource is managed-as-code (editable:false);
set PROMETHEUS_IS_DEFAULT=true only if this Grafana has no other default. For an
existing host Grafana, copy grafana/provisioning-production into /etc/grafana/provisioning
and set the same env vars. Put Grafana behind TLS; never expose :3000 publicly.

## Security

See SECURITY.md. Read-only observer; no validator keys. Keep telemetry and
metrics bound to loopback; never run the demo stack on a validator host.

## License

MIT. See LICENSE.