# Security Policy

## What this is
A read-only observer. It makes unauthenticated GET requests to your local
Optimum Gateway telemetry port and re-publishes the results as Prometheus
metrics. It never touches validator keys, never signs anything, and makes no
outbound calls beyond the gateway. Blast radius is low by design.

## Reporting a vulnerability
Please report privately via GitHub: Security tab -> "Report a vulnerability"
(enable private vulnerability reporting in repo Settings first). Do not open
public issues for security reports. We aim to acknowledge within 7 days.

## Supported versions
Only the latest `main` is supported; pin a specific
commit (or image digest) for any real deployment and review the code yourself.

## Deployment guidance (the real risks)
- The gateway telemetry port is UNAUTHENTICATED. Keep it bound to 127.0.0.1
  (`telemetry_addr`) and never expose it publicly.
- Bind exporter metrics to loopback or your monitoring VLAN, not 0.0.0.0.
- Images are upstream stock, pinned by tag in compose. For fleet rollouts, verify digests and canary on one host.
- Do NOT run the demo compose (admin/admin Grafana) on a validator host.
- Alert on `probe_success` / `probe_http_status_code` (down vs degraded), not on a stale health gauge.

## Out of scope
Vulnerabilities in the Optimum Gateway itself belong to Optimum, not this repo.
Report those to the Optimum team per their SECURITY.md.