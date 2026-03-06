# Skill: Database (Atlas + SQL + sqlc + pgx)

## Purpose
Keep database evolution, queries, and runtime access reproducible and reviewable.

## SSOT
- Schema design: `docs/01_architecture/DB_SCHEMA_DESIGN.md`
- Migration flow: `docs/05_operations/MIGRATION_FLOW.md`
- sqlc rules: `docs/02_tech_stack/SQLC_QUERY_RULES.md`

## Rules (non-negotiable)
- SQL is the source of truth for data access.
- Do not introduce ORMs.
- Never manually edit generated sqlc outputs.

## Safe defaults
- Prefer `NOT NULL` + defaults; use nullable types only when required.
- Prefer expand/contract migrations for safe deploy/rollback.
- Keep queries small, indexed, and explainable.

## Common mistakes to avoid
- Writing “SELECT *” in sqlc queries.
- Mixing business logic into repository layer.
- Introducing breaking schema changes without a multi-step rollout.

## Checklist
- Schema change reflected in `schema.hcl` first.
- Queries added/updated in `sql/queries/*.sql`.
- Regenerate (`sqlc generate`) and keep diffs small.
- Migration plan includes backward compatibility and rollback path.
