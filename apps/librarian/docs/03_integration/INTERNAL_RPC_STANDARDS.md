# INTERNAL_RPC_STANDARDS

本サービス（Librarian）は Professor と **gRPC/Proto（双方向ストリーミング）** で連携する。

契約の SSOT は Professor 側の `proto/librarian/v1/librarian.proto`。
内部 RPC の標準（IDL、コード生成、互換性ルール等）は Professor 側の SSOT（`PROTOBUF_GRPC_STANDARDS.md`）を正とする。

## 関連
- `03_integration/INTER_SERVICE_COMM.md`
- `03_integration/API_CONTRACT_WORKFLOW.md`
- Professor 側の `PROTOBUF_GRPC_STANDARDS.md`
