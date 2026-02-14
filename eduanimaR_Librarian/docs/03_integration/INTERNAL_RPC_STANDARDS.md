# INTERNAL_RPC_STANDARDS（Not Applicable）

本サービス（Librarian）は Professor と **HTTP/JSON（OpenAPI）** で連携する。
そのため、HTTP/JSON 以外の内部 RPC 方式に関する標準（IDL、コード生成、互換性ルール等）は、
本サービスの SSOT としては採用しない。

必要な場合は Professor 側（Go）の SSOT を正とする。

## 関連
- `03_integration/INTER_SERVICE_COMM.md`
- `03_integration/API_CONTRACT_WORKFLOW.md`
