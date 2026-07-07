# APSystems Prometheus Exporter

Prometheus exporter for [APSystems](https://www.apsystems.com) micro-inverter systems via the official **OpenAPI** (EMA cloud). Exposes **per-panel** energy and power metrics for Grafana dashboards, plus whole-system totals.

Designed as a **standalone open-source project** — copy or clone this directory to its own repository. No cluster-specific configuration is included.

## Features

- Per-panel metrics: `apsystems_panel_energy_today_kwh`, `apsystems_panel_power_watts`
- System totals and ECU health status
- Optional panel layout file for `array` / `row` / `col` labels (roof grid dashboards)
- Rate-limit friendly polling (configurable interval + solar-hours window)
- Helm chart with optional Prometheus Operator `ServiceMonitor`
- Distroless container image

## Metrics

| Metric | Labels | Description |
|--------|--------|-------------|
| `apsystems_panel_energy_today_kwh` | uid, channel, array, row, col | Cumulative kWh today per panel channel |
| `apsystems_panel_power_watts` | uid, channel, array, row, col | Latest power sample (W) per panel |
| `apsystems_system_energy_today_kwh` | sid | System total kWh today |
| `apsystems_system_energy_month_kwh` | sid | System total kWh this month |
| `apsystems_system_energy_year_kwh` | sid | System total kWh this calendar year |
| `apsystems_system_energy_lifetime_kwh` | sid | System total kWh since install |
| `apsystems_system_status` | sid | 1=green 2=yellow 3=red 4=grey |
| `apsystems_exporter_scrape_success` | — | 1 on successful scrape |
| `apsystems_exporter_api_calls_total` | — | API call counter |
| `apsystems_exporter_last_scrape_timestamp_seconds` | — | Last successful scrape |

## Prerequisites

1. **OpenAPI credentials** from APSystems (email support or EMA portal → Settings → OpenAPI Service).
2. **System ID (`sid`)** and **ECU ID (`eid`)** from the EMA portal (Report → System Data → ECU Data).
3. LV0 free tier is **1,000 API calls/month** — default scrape settings target ~480–600/month.

## Quick start (binary)

```bash
export APS_APP_ID=your-32-char-app-id
export APS_APP_SECRET=your-12-char-secret
export APS_SID=your-system-id
export APS_ECU_ID=your-ecu-id
export APS_PANELS_LAYOUT=examples/panels-layout.json  # optional
export TZ=America/Edmonton
export SCRAPE_INTERVAL=2h

go run ./cmd/apsystems-exporter
curl localhost:9921/metrics
```

## Docker

```bash
docker build -t apsystems-exporter .
docker run --rm -p 9921:9921 \
  -e APS_APP_ID -e APS_APP_SECRET -e APS_SID -e APS_ECU_ID \
  apsystems-exporter
```

Images are published to:

- **GitHub Container Registry:** `ghcr.io/denislemire/apsystems-prometheus-exporter` (GitHub Actions on `v*` tags)
- **EhWS internal registry:** `registry.ehws.generic.business/apsystems-exporter` (CircleCI Server, KubeVirt `linux.medium` machine executor)

## Helm

```bash
helm install apsystems ./helm/apsystems-exporter \
  --namespace monitoring --create-namespace \
  --set apsystems.sid=YOUR_SID \
  --set apsystems.ecuId=YOUR_ECU_ID \
  --set apsystems.existingSecret=apsystems-openapi-credentials \
  --set scrape.timezone=America/Edmonton \
  --set serviceMonitor.enabled=true \
  --set serviceMonitor.additionalLabels.release=monitoring
```

Secret keys expected: `app-id`, `app-secret`.

### Panel layout (Grafana roof grid)

Map each `uid-channel` to grid coordinates. Copy positions from the EMA **MODULE** tab (row/col) or your aerial layout:

```yaml
panelsLayout:
  panels:
    "804000060846-1":
      uid: "804000060846"
      channel: 1
      array: west
      row: 0
      col: 2
```

See [examples/panels-layout.json](examples/panels-layout.json).

### Grafana query examples

```promql
# All panels today (kWh)
apsystems_panel_energy_today_kwh

# West array only
apsystems_panel_energy_today_kwh{array="west"}

# Compare to Sense whole-array solar (separate exporter)
max(sense_energy_power_watts{name="solar"})
```

Build a **Stat** or **Bar gauge** panel per `uid`/`channel`, positioned by `row`/`col` labels using Grafana's grid or a canvas plugin.

## Configuration

| Env / flag | Default | Description |
|------------|---------|-------------|
| `APS_APP_ID` | — | OpenAPI App ID (required) |
| `APS_APP_SECRET` | — | OpenAPI App Secret (required) |
| `APS_SID` | — | System ID (required) |
| `APS_ECU_ID` | — | ECU ID (required) |
| `APS_API_BASE` | `https://api.apsystemsema.com:9282` | API base URL |
| `SCRAPE_INTERVAL` | `2h` | Poll interval (`2h`, `3600`, etc.) |
| `TZ` | `UTC` | Timezone for date + solar window |
| `SOLAR_START_HOUR` | `6` | Start polling (local hour) |
| `SOLAR_END_HOUR` | `22` | Stop polling (local hour) |
| `SUMMARY_EVERY_N` | `6` | System summary every N scrapes |
| `APS_PANELS_LAYOUT` | `/etc/apsystems/panels-layout.json` | Panel layout JSON |
| `EXPORTER_LISTEN` | `:9921` | HTTP listen address |

## API quota

Each scrape uses **2 calls** (batch energy + batch power). Summary adds **2 calls** every `SUMMARY_EVERY_N` scrapes.

Example: `SCRAPE_INTERVAL=2h`, solar window 16h → 8 scrapes/day × 2 = **16 calls/day** (~480/month).

## Publishing this repository

This folder is intended to live at:

**https://github.com/denislemire/apsystems-prometheus-exporter**

```bash
cd contrib/apsystems-prometheus-exporter
git init
git remote add origin git@github.com:denislemire/apsystems-prometheus-exporter.git
git add .
git commit -m "Initial release"
git tag v0.1.0
git push -u origin main --tags
```

GitHub Actions builds and pushes to GHCR on tag (`v*`). EhWS CircleCI builds and pushes to Zot on the same tags (and branch pushes for CI validation).

### EhWS CircleCI (Zot)

Requires a CircleCI project on `circle.ehws.generic.business` with project env var `OP_SERVICE_ACCOUNT_TOKEN` (1Password service account — not committed). The job installs 1Password CLI and reads Zot credentials via `op://` references; see `ehws-infra` → `docs/CIRCLECI_1PASSWORD_SECRETS.md` for setup.

## License

MIT — see [LICENSE](LICENSE).

## References

- APSystems OpenAPI user manual (request from APSystems support)
- Signature: HMAC-SHA256 over `{timestamp}/{nonce}/{appId}/{lastPathSegment}/{METHOD}/HmacSHA256`
