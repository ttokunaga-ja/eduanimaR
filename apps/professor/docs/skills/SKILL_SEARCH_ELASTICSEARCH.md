# Skill: Search (Elasticsearch)

## Purpose
Keep search behavior correct, observable, and safely synchronized with the system of record.

## SSOT
- Elasticsearch ops: `docs/02_tech_stack/ELASTICSEARCH_OPS.md`
- Sync strategy: `docs/01_architecture/SYNC_STRATEGY.md`

## Safe defaults
- Version mappings and treat mapping changes as migrations.
- Prefer read patterns that are explainable (avoid accidental heavy aggregations).
- Design for eventual consistency: define freshness expectations and user impact.

## Common mistakes to avoid
- Changing mappings without a reindex/backfill plan.
- Treating Elasticsearch as the system of record.
- Hidden query complexity that makes p95/p99 unpredictable.

## Checklist
- Mapping/index changes have a rollout and rollback plan.
- Sync/backfill behavior is monitored and alertable.
- Query changes include performance validation.
