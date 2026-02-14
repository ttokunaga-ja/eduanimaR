# Skill: Observability (OpenTelemetry + SLO)

## Purpose
Make the system debuggable and operable in production.

## SSOT
- Observability: `docs/05_operations/OBSERVABILITY.md`
- SLO/alerting: `docs/05_operations/SLO_ALERTING.md`

## Safe defaults
- Correlate logs/traces/metrics with consistent IDs (`request_id`, `trace_id`).
- Instrument boundaries (HTTP/gRPC handlers, outbound clients, DB, queue).
- Prefer dashboards + alerts derived from SLOs.

## Common mistakes to avoid
- Logging unstructured blobs and expecting search to save you.
- High-cardinality labels in metrics.
- Alerts on “symptoms” (CPU) instead of “user impact” (error rate/latency).

## Checklist
- Each request path has trace spans and key attributes.
- Key services have golden signals (latency, traffic, errors, saturation).
- Alerts have runbooks and clear owner.
