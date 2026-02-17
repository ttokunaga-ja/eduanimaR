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

## 3) Phase 1開始前の必須作業

### 3.1 OpenAPI定義の初期化（MUST）
1. `docs/openapi.yaml`が以下のエンドポイントを定義していることを確認:
   - `POST /v1/subjects/{subjectId}/materials` (資料アップロード)
   - `POST /v1/qa/stream` (SSE応答)
   - `POST /v1/qa/feedback` (フィードバック送信)
   - `GET /v1/subjects` (科目一覧、Web版固有)
   - `GET /v1/subjects/{subject_id}/materials` (資料一覧、Web版固有)
   - `GET /v1/subjects/{subject_id}/conversations` (会話履歴、Web版固有)
   - `GET /v1/materials/{materialId}/status` (処理状態確認)

2. 定義が不足している場合は、`docs/openapi.yaml`を上記仕様に従って作成する

3. **Web版固有機能の必須実装**:
   - 科目一覧取得API（トップメニューバーのプルダウン用）
   - 資料一覧取得API（選択科目の資料一覧表示用）
   - 会話履歴取得API（選択科目の会話履歴表示用）

### 3.2 契約テストの準備（MUST）
1. `internal/contracttest/ssot_test.go`で以下を検証:
   - `docs/openapi.yaml`の存在
   - `proto/librarian/v1/librarian.proto`の存在
   - 生成コードとの整合性

2. CI で `contract-codegen-check` が実行されることを確認（`05_operations/CI_CD.md`参照）

### 3.3 DB Schema の初期化（推奨）
1. `docs/01_architecture/DB_SCHEMA_DESIGN.md`にER図が記載されていることを確認
2. Atlas によるマイグレーション準備:
   ```bash
   atlas schema apply --env local --auto-approve
   ```

3. **Phase 1→Phase 2移行時のDB変更準備**:
   - `users`テーブルへ `provider`, `provider_user_id` カラム追加準備
   - SSO認証用のセッションテーブル追加準備
