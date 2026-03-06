# CODEGEN_OWNERSHIP (Librarian)

## SSOT
- Proto contract source: `contracts/proto/librarian/v1/librarian.proto`

## Generated Output (Current)
- Path: `apps/librarian/src/librarian/v1/`
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
- Hand-written code should stay under `src/librarian/` (except `src/librarian/v1/*`).

## Lint/Test Rule
- Lint target excludes generated stubs to avoid non-actionable style errors.
