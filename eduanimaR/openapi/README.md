# OpenAPI (Project SSOT)

Place the project contract here as `openapi/openapi.yaml` (or update `orval.config.ts`).

Minimum workflow:
- Update/obtain OpenAPI
- Run `npm run api:generate`
- Commit generated changes under `src/shared/api/generated`

---

## OpenAPI定義の配置（Phase 1開始前に必須）

### 契約の配置場所
- **SSOT**: `eduanimaR_Professor/docs/openapi.yaml`
  - Professor が管理する外向きAPI定義
- **生成先**: `eduanimaR/src/shared/api/generated/`
  - Orval で自動生成されるクライアントコード

### Phase 1開始条件
1. Professor 側で `docs/openapi.yaml` に以下のエンドポイントが定義されていること:
   - `POST /v1/auth/dev-login` (Phase 1のみ・dev-user自動発行)
   - `POST /v1/qa/stream` (SSEでQ&A応答)
   - `GET /v1/subjects` (科目一覧取得)
   - `GET /v1/subjects/{subject_id}/materials` (資料一覧取得)

2. Orval 設定（`orval.config.ts`）が上記定義を読み込めること

### 生成コマンド
```bash
npm run api:generate
```

### CI要件
- `contract-codegen-check` で差分を検出（`docs/05_operations/CI_CD.md`参照）
