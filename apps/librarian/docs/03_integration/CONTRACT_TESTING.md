# CONTRACT_TESTING

## 目的
契約（OpenAPI）が破壊されるのを、レビューの目視だけに頼らず **CI で機械的に検出**する。

## 対象
- サービス間契約: OpenAPI（Professor ↔ Librarian）

## 原則（MUST）
- 契約は SSOT から生成し、生成物の手編集を禁止
- “互換性” を breaking check で自動判定する
- 破壊的変更は、バージョン分離/移行期間/告知までセットで扱う

## OpenAPI
### 1) 生成差分検出（MUST）
- `openapi.yaml` の再生成で差分が出ないこと
  - 目的: 手編集や生成漏れの検出

### 2) Breaking change 検出（SHOULD）
- `main`（または前回リリース）と比較して breaking を検出
- 破壊的変更は、`API_VERSIONING_DEPRECATION.md` の手順に従う

### 3) 契約駆動のスモーク（推奨）
- OpenAPI からリクエストを生成し、
  - 200系
  - 4xx（バリデーション）
  - 5xx（想定外）
  を最低限確認する

## CI への組み込み（例）
- Stage: Contract
  - OpenAPI: 再生成差分 + breaking check

> 具体ツール（buf / openapi-diff / schema registry 等）はプロジェクトで決める。
> 本テンプレは“何を検査すべきか”を標準化する。

## 関連
- `03_integration/API_CONTRACT_WORKFLOW.md`
- `03_integration/API_VERSIONING_DEPRECATION.md`
- `05_operations/CI_CD.md`
