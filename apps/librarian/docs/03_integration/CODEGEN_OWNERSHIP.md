# CODEGEN_OWNERSHIP (Librarian)

## SSOT
- Proto contract source: `contracts/proto/librarian/v1/librarian.proto`

## Generated Output (Current)
- Path: `apps/librarian/gen/proto/python/librarian/v1/`
- Files:
  - `librarian_pb2.py`
  - `librarian_pb2.pyi`
  - `librarian_pb2_grpc.py`
  - `librarian_pb2_grpc.pyi`

## Generate Command
```bash
cd apps/librarian
make proto
```

## Ownership Rule
- Do not hand-edit generated files.
- Regenerate from SSOT contract after proto changes.
- Hand-written code should stay under `src/librarian/`.
- Runtime import shims under `src/librarian/v1/` must stay thin and generated-file agnostic.
- Generated stubs under `gen/proto/python/librarian/v1/` are tracked in Git for reproducible Docker/CI builds.

## Lint/Test Rule
- Lint target excludes generated stubs to avoid non-actionable style errors.
