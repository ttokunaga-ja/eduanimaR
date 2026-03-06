# Skill: Supply-Chain Security (SLSA-aligned)

## Purpose
Reduce the risk of compromised dependencies, build steps, and artifacts.

## SSOT
- Supply-chain security: `docs/05_operations/SUPPLY_CHAIN_SECURITY.md`
- CI/CD: `docs/05_operations/CI_CD.md`

## Safe defaults
- Pin tool/codegen versions.
- Generate SBOMs for released artifacts (or at least container images).
- Prefer least-privilege CI tokens.

## Common mistakes to avoid
- Floating versions for build/codegen tools.
- Committing binaries/generated artifacts that hide malicious changes.
- Broad dependency bumps without review focus.

## Checklist
- CI runs vuln scanning and produces SBOM/provenance as defined.
- Secrets are injected via secret managers; never committed.
- Release process is reproducible and reviewable.
