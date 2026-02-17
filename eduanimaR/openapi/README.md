# OpenAPI (Project SSOT)

## ⚠️ 同期ルール（必読）

`eduanimaR/openapi/openapi.yaml` は **`eduanimaR_Professor/docs/openapi.yaml` と常に同一内容を維持する**。

- **契約の正（SSOT）**: `eduanimaR_Professor/docs/openapi.yaml`（Professor チームが管理）
- **フロントエンド参照コピー**: `eduanimaR/openapi/openapi.yaml`（Orval がこのパスを読む）
- **同期タイミング**: Professor 側の openapi.yaml を更新した際は、必ず本ファイルも同一内容で更新すること
- **差分検出**: CI の `contract-codegen-check` で openapi.yaml → 生成コードの差分を検出する

```
SSOT: eduanimaR_Professor/docs/openapi.yaml
  ↓ 手動コピー（変更時に同期）
  eduanimaR/openapi/openapi.yaml
  ↓ npm run api:generate（Orval）
  eduanimaR/src/shared/api/generated/
```

## 最短ワークフロー
1. Professor 側で `eduanimaR_Professor/docs/openapi.yaml` を更新
2. 本ファイル（`eduanimaR/openapi/openapi.yaml`）を同一内容で更新
3. `npm run api:generate` を実行
4. 生成変更（`src/shared/api/generated`）をコミット

---

## OpenAPI定義の配置（Phase 1開始前に必須）

### 契約の配置場所
- **SSOT**: `eduanimaR_Professor/docs/openapi.yaml`
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
