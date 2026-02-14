# API_GEN

## 目的
本リポジトリ（バックエンド）で定義した OpenAPI 契約を前提に、フロントエンド（Next.js + TypeScript）が **Orval** を使って安全に型生成できる状態を維持する。

## 原則
- OpenAPI が正（`API_CONTRACT_WORKFLOW.md`）
- 生成の手順/成果物のコミット方針は、フロントエンド側の方針と揃える
- **手書きの型定義・fetch 関数を禁止** し、生成に統一する

## フロントエンド側の生成ツール（確定）
- **Orval** (推奨): OpenAPI から TypeScript 型 + React Query Hooks を生成
- または **OpenAPI Generator**: 汎用的だが、Orval の方が React Query 統合が優れている

## バックエンド側の責務
1. 常に最新の `openapi.yaml` (または JSON) をエクスポートする
2. 変更時は以下を明記する：
   - **Breaking Changes**: 既存クライアントが壊れる変更（必須フィールド追加、型変更、エンドポイント削除等）
   - **Compatible Changes**: 後方互換（任意フィールド追加、新エンドポイント等）
3. エラーコードは `ERROR_CODES.md` をSSOTとして、フロントエンドと同期する

## 受け手（フロントエンド）との連携
- バックエンドが `openapi.yaml` を更新したら、フロントエンドは `npm run api:generate` を実行
- 生成されたコードは `src/shared/api/` に配置され、FSD の上位層（entities/features）から import される
- **型定義の手書きは禁止**。すべて生成物を使う。

## 配置例（フロントエンド側）
```
src/shared/api/
├── user.gen.ts       # Orval生成: User型 + useGetUser() Hook
├── product.gen.ts    # Orval生成: Product型 + useGetProducts() Hook
└── order.gen.ts      # Orval生成: Order型 + useCreateOrder() Hook
```

