# Quickstart（Microservices Template）

目的：テンプレ取り込み直後に、サービス境界・契約・運用の「開始条件」を短時間で確定させる。

## 0) 開始条件（Must）
- サービス境界（責務/依存/公開IF）が `01_architecture/MICROSERVICES_MAP.md` に反映されている
- 外向き契約（OpenAPI）/ 内向き契約（Proto）がどちらも SSOT として場所が決まっている
- CI の最低ゲート（lint/test/build/contract drift）が `05_operations/CI_CD.md` の方針で組める

## 1) 最短で読む順（推奨）
1. `02_tech_stack/STACK.md`
2. `01_architecture/MICROSERVICES_MAP.md`
3. `01_architecture/CLEAN_ARCHITECTURE.md`
4. `03_integration/API_CONTRACT_WORKFLOW.md`
5. `03_integration/PROTOBUF_GRPC_STANDARDS.md`
6. `05_operations/RELEASE_DEPLOY.md`

## 2) まず埋める（プロジェクト固有）
- `00_quickstart/PROJECT_DECISIONS.md`
