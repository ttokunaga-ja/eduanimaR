# SKILL: Orval / OpenAPI（契約駆動生成）

対象：API契約ズレを根絶し、手書き型/手書きfetchを排除する。

変化に敏感な領域：
- OpenAPI の変更（互換性）
- 生成設定（operationId等）の変更

関連：
- `../03_integration/API_GEN.md`
- `../03_integration/API_CONTRACT_WORKFLOW.md`
- `../03_integration/API_VERSIONING_DEPRECATION.md`

---

## Versions（2026-02-11 / dist-tag: latest）

- `orval`: `8.2.0`

---

## Must
- APIクライアント/型は生成物をSSOTにする
- `generated/` は手編集禁止
- 破壊的変更は廃止手順に従う

### 実装メモ（導入手順の最小）

- OpenAPI を取得 → `npm run api:generate` で `src/shared/api/generated` を更新
- 手書き拡張（baseURL/認証/エラー正規化）は `client.ts` / `errors.ts` に閉じ込める

## 禁止
- 画面内の手書き `fetch/axios`
- OpenAPI に無い仕様の推測実装

## チェックリスト
- [ ] OpenAPI 変更は互換か？（互換でないなら廃止手順）
- [ ] `npm run api:generate` を回したか？CIで差分検知されるか？
- [ ] 生成物を直接修正していないか？
