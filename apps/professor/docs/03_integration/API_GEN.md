# API_GEN

## 目的
本リポジトリ（Professor: Go）で定義した OpenAPI 契約を前提に、consumer（例: フロントエンド）がクライアント/型生成ツールを使って安全に統合できる状態を維持する。

## 原則
- OpenAPI が正（`API_CONTRACT_WORKFLOW.md`）
- 生成の手順/成果物のコミット方針は、フロントエンド側の方針と揃える
- **手書きの型定義・fetch 関数を禁止** し、生成に統一する

## フロントエンド側の生成ツール（確定）
consumer 側で OpenAPI からクライアント/型生成を行う場合がある（ツールは consumer 側の裁量）。

## バックエンド側の責務
1. 常に最新の `docs/openapi.yaml` を SSOT として維持する
2. 変更時は以下を明記する：
   - **Breaking Changes**: 既存クライアントが壊れる変更（必須フィールド追加、型変更、エンドポイント削除等）
   - **Compatible Changes**: 後方互換（任意フィールド追加、新エンドポイント等）
3. エラーコードは `ERROR_CODES.md` をSSOTとして、フロントエンドと同期する

## 受け手（フロントエンド）との連携
Professor 側が `docs/openapi.yaml` を更新したら、consumer 側は必要に応じてクライアント/型生成を再実行する。
- 生成されたコードは `src/shared/api/` に配置され、FSD の上位層（entities/features）から import される

