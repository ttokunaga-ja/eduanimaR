# Skill: Resiliency (Timeouts, Retries, Idempotency)

## Purpose
Avoid cascading failures and make partial outages survivable.

## SSOT
- Resiliency rules: `docs/01_architecture/RESILIENCY.md`
- Inter-service comm: `docs/03_integration/INTER_SERVICE_COMM.md`
- Sync strategy (DB ↔ Search / CDC): `docs/01_architecture/SYNC_STRATEGY.md`

## Safe defaults
- Timeouts first: set deadlines at the boundary and propagate via context.
- Retry only idempotent operations and only where you can bound cost.
- Prefer explicit idempotency keys for externally triggered mutations.

## Common mistakes to avoid
- Unbounded retries or “retry everywhere”.
- Retrying non-idempotent operations without deduplication.
- Ignoring queue/backpressure signals.

## Checklist
- Each network hop has a timeout budget.
- Retries are single-layer with jitter/backoff and max attempts.
- Idempotency strategy is documented and testable.
