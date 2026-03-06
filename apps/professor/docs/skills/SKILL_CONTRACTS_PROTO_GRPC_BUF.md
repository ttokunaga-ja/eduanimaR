# Skill: Contracts (Proto + gRPC + Buf)

## Purpose
Prevent contract drift and production breakages in internal service communication.

## SSOT
- gRPC/Proto standards: `docs/03_integration/PROTOBUF_GRPC_STANDARDS.md`
- Inter-service comm rules: `docs/03_integration/INTER_SERVICE_COMM.md`
- Error mapping: `docs/03_integration/ERROR_HANDLING.md`

## Safe defaults
- `.proto` is the SSOT for internal APIs.
- Backward compatibility by default: never reuse field numbers; prefer additive changes.
- Every RPC must have a deadline; never allow unbounded waits.

## Operational essentials
- Use interceptors for auth/logging/tracing/deadline enforcement.
- Implement health checking (`grpc.health.v1.Health`) where applicable.

## Common mistakes to avoid
- Breaking changes without explicit versioning.
- Adding retries at multiple layers (client + gateway + usecase) without budgets.
- Ignoring cancellation and continuing DB work after the caller is gone.

## Checklist
- Run `buf lint` / `buf breaking` (or ensure CI covers it).
- Verify deadline propagation and cancellation behavior.
- Verify status-code mapping is consistent with `ERROR_HANDLING.md`.
