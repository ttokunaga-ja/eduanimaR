# Skills Portal

This folder contains **Skill** documents: compact, high-signal “rules of engagement” for Librarian.

## How to use
- Before proposing or implementing changes, open the relevant Skill doc(s) and follow the **Safe defaults** and **Checklists**.
- If you need to introduce a new tool/framework or a new architectural pattern, first update:
  1) `docs/02_tech_stack/STACK.md` (the stack decision)
  2) the relevant Skill doc(s) here (to prevent future drift)

## Index
- Stack & SSOT: `SKILL_STACK_SSOT.md`
- Contracts (OpenAPI): `SKILL_CONTRACTS_OPENAPI.md`
- Resiliency: `SKILL_RESILIENCY_TIMEOUTS_RETRIES_IDEMPOTENCY.md`
- Observability (OTel/SLO): `SKILL_OBSERVABILITY_OTEL_SLO.md`
- API security (OWASP): `SKILL_API_SECURITY_OWASP.md`
- Supply-chain security (SLSA/SBOM): `SKILL_SUPPLY_CHAIN_SLSA_SBOM.md`
- Deploy (GCP Cloud Run): `SKILL_DEPLOY_GCP_CLOUD_RUN.md`

## Not applicable (template leftovers)
The following docs are intentionally stubbed for Librarian, because this service is DB-less and uses HTTP/OpenAPI as SSOT:
- `SKILL_GO_1_25_BACKEND.md`
- `SKILL_DATABASE_BOUNDARY.md`
- `SKILL_SEARCH_BOUNDARY.md`
- `SKILL_EVENTS_BOUNDARY.md`
- `SKILL_CONTRACTS_INTERNAL_RPC.md`
