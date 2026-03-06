# Skill: Events / CDC / Kafka

## Purpose
Make event-driven integrations reliable and evolvable without breaking consumers.

## SSOT
- Event contracts: `docs/03_integration/EVENT_CONTRACTS.md`
- Sync strategy: `docs/01_architecture/SYNC_STRATEGY.md`

## Safe defaults
- Define event schemas and compatibility rules.
- Choose partition keys intentionally (ordering and scaling).
- Make consumers idempotent and support replay.

## Common mistakes to avoid
- Using events as “RPC over Kafka”.
- No DLQ/reprocessing story.
- Non-idempotent consumers that break under retries/replays.

## Checklist
- Compatibility strategy is documented (additive changes by default).
- DLQ and reprocessing procedures exist.
- Producers and consumers have clear ownership and oncall escalation paths.
