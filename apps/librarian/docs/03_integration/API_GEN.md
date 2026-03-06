# API_GEN

## 目的
本リポジトリ（Librarian）で定義した OpenAPI 契約を前提に、Professor（Go）が安全にクライアント生成・統合できる状態を維持する。

## 原則
- OpenAPI が正（`API_CONTRACT_WORKFLOW.md`）
- 生成の手順/成果物のコミット方針は、Professor 側の方針と揃える
- 手書きの HTTP クライアント実装を乱立させない（生成 or 単一ラッパに統一）

## Professor 側の生成ツール（例）
- OpenAPI Generator（言語問わず）
- Go の生成ツール（例: oapi-codegen）

> どれを正とするかは Professor 側 SSOT で固定し、このドキュメントには「契約は OpenAPI が正」であることだけを残す。

## Librarian 側の責務
1. 常に最新の `openapi.yaml` (または JSON) をエクスポートする
2. 変更時は以下を明記する：
   - **Breaking Changes**: 既存クライアントが壊れる変更（必須フィールド追加、型変更、エンドポイント削除等）
   - **Compatible Changes**: 後方互換（任意フィールド追加、新エンドポイント等）
3. エラーコードは `ERROR_CODES.md` をSSOTとして、フロントエンドと同期する

## 受け手（Professor）との連携
- Librarian が `openapi.yaml` を更新したら、Professor 側でクライアント生成/更新を行う
- 生成物は Professor 側の方針に従う（手書きクライアントの乱立を避ける）

## 配置例
（Professor 側のSSOTに従う）

