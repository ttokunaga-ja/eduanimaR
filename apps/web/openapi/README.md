# openapi/ — このディレクトリについて

OpenAPI 定義の SSOT は **`contracts/openapi/professor.yaml`** に移動しました。

- **Orval** は `orval.config.ts` で `../../contracts/openapi/professor.yaml` を直接参照しています。
- このディレクトリには `openapi.yaml` のコピーを置きません（コピーは contract drift の原因になるため）。

## 生成ワークフロー

```
contracts/openapi/professor.yaml   ← SSOT（ここを編集する）
  ↓ npm run api:generate（Orval）
  src/shared/api/generated/        ← 生成コード（git追跡対象外）
```

契約変更手順:
1. `contracts/openapi/professor.yaml` を更新
2. `npm run api:generate` を実行
3. 生成差分を PR に含める（CI の `contract-drift` が自動検証する）

- **SSOT**: `contracts/openapi/professor.yaml`
  - Professor が管理する外向きAPI定義（変更はここに加える）
- **フロントエンドコピー**: `eduanimaR/openapi/openapi.yaml`
  - SSOT と常に同一内容。Orval の入力ファイル
- **生成先**: `eduanimaR/src/shared/api/generated/`
  - Orval で自動生成されるクライアントコード

### Phase 1開始条件
1. Professor 側で `docs/openapi.yaml` に以下のエンドポイントが定義されていること:

| エンドポイント | 用途 |
|---|---|
| `POST /v1/auth/dev-login` | Phase 1固定ユーザーログイン（dev-user） |
| `GET /v1/subjects` | 科目一覧（`?lms_course_id=`で拡張機能コース判別にも使用） |
| `POST /v1/subjects` | 科目作成 |
| `GET /v1/subjects/{subject_id}/materials` | 資料一覧（Web版「資料一覧」表示） |
| `POST /v1/subjects/{subject_id}/materials` | 資料アップロード（202 Accepted、Kafka非同期） |
| `GET /v1/subjects/{subject_id}/materials/{material_id}` | 処理状態ポーリング |
| `POST /v1/subjects/{subject_id}/chats` | 質問送信（SSEストリーミング） |
| `GET /v1/subjects/{subject_id}/chats` | 会話履歴一覧（Web版「会話履歴」表示） |
| `GET /v1/subjects/{subject_id}/chats/{chat_id}` | 会話詳細 |
| `POST /v1/subjects/{subject_id}/chats/{chat_id}/feedback` | Good/Bad フィードバック |
| `GET /healthz` / `GET /readyz` | ヘルスチェック |

2. Orval 設定（`orval.config.ts`）が上記定義を読み込めること

### 生成コマンド
```bash
npm run api:generate
```

### CI要件
- `contract-codegen-check` で差分を検出（`docs/05_operations/CI_CD.md`参照）
