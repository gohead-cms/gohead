# Observability

## Health

`GET /_health` → `{"status":"ok"}` (no authentication). Use it for liveness and readiness
probes. The admin UI image answers on `/healthz`.

## Metrics

`GET /_metrics` exposes Prometheus metrics, including:

| Metric | Type | Labels |
|--------|------|--------|
| `http_requests_total` | counter | `code`, `method` |
| `http_request_duration_seconds` | histogram | `code`, `method` |

plus the standard Go and process collectors. Scrape the API pods on port 8080. Do not expose
`/_metrics` publicly: restrict it at your ingress.

## Logs

All processes log JSON lines to stdout, with `level`, `msg`, `time` and contextual fields
(`collection`, `agent_name`, `item_id`, …). Set the level with `GOHEAD_LOG_LEVEL`
(`debug` also logs SQL statements and agent conversations; avoid it in production).

## Tracing

Set `GOHEAD_TELEMETRY_ENABLED=true` to export OpenTelemetry traces of HTTP requests over
OTLP/gRPC. Configure the destination with the standard variables:

```bash
GOHEAD_TELEMETRY_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector.observability:4317
OTEL_EXPORTER_OTLP_INSECURE=true
```

Spans use the service name `gohead` and are flushed on shutdown. A minimal collector config is
in `otel-collector-config.yaml`.

## Queues

Redis queues are managed by [Asynq](https://github.com/hibiken/asynq): `events` (content events,
read by the dispatcher) and `agents` (agent runs, read by workers). Inspect them with the
`asynq` CLI or Asynqmon:

```bash
asynq --uri redis.internal:6379 stats
```
