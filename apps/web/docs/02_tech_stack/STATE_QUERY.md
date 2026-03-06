# State Management & Data Fetching Policy

このドキュメントは「状態管理」と「データ取得」の責務分離を固定し、AI が場当たり的に状態管理ライブラリや `fetch` を混ぜないための制約です。

## 前提（確定スタック）
本テンプレートの確定版技術スタックは [STACK.md](./STACK.md) を参照。

前提ツール：
- サーバー状態：TanStack Query
- APIクライアント：OpenAPI から生成（Orval）
- フォーム：React Hook Form + Zod

## 基本方針
- サーバーデータ取得は原則 `src/shared/api` の自動生成（Orval）を使用
	- Client：生成 hooks（TanStack Query）
	- Server（RSC/Route Handler）：生成クライアント（非hook）
- UI 状態（フォーム入力、開閉など）はコンポーネントローカル state を基本
- グローバル状態は最小化し、導入する場合は `src/app/providers` に集約

## 禁止
- コンポーネント内での手書き `fetch` / `axios`（生成物がある前提）
- `features` 間の直接依存を作るためのグローバル store 乱用
- Server/Client の境界を無視して「なんとなく全部 client」にする

## 使い分け（運用テンプレ）

### 1) サーバー状態（TanStack Query）
対象：バックエンド由来のデータ（一覧、詳細、ユーザー、設定など）

- 取得：生成 `useXxxQuery` / `useSuspenseXxxQuery`（生成方式に合わせる）
- 更新：生成 `useXxxMutation`
- 反映：原則は `invalidateQueries`（正しさ優先）。必要に応じて `setQueryData` で部分更新

#### Query Key の規約
- queryKey は「リソース + パラメータ」が分かる形にする（例：`['user', userId]`）
- “画面名” を queryKey に混ぜない（再利用しづらくなる）
- queryKey を組み立てる関数は slice の `model` か `shared/lib` に寄せ、ばらまかない

#### Mutation の規約
- mutation から別feature/entityへ直接依存しない
- 成功後のキャッシュ整合は「どの query を invalidation するか」を明示する

#### エラー/認可の扱い
- API エラー分類（401/403/422/500など）は `shared/api` 側で統一し、UI側は分類結果で分岐する
- “ログインに飛ばす” のような制御は `pages` / `app` 側の合成点で行う（feature内でルーティングを抱え込まない）

### 2) UI 状態（Client state）
対象：フォーム、モーダル開閉、タブ選択、入力途中の値、並び替え等

- 基本：`useState` / `useReducer`
- フォーム：React Hook Form + Zod（`features/*` に配置）
- 共有が必要：まず `widgets` / `pages` で合成して解決できないか検討

### 3) グローバル状態（最小化）
対象：テーマ、言語、セッションなど“アプリ全体”の関心事

- 置き場：`src/app/providers`
- 導入判断：
	- 画面をまたぐ必要がある
	- URL（search params）に置くのが不適
	- TanStack Query（サーバー状態）では表現できない

---

## Next.js App Router（RSC）との統合ポリシー

- 可能な限り Server Component で静的/準静的に描画し、操作が必要なところだけ `"use client"`
- Client コンポーネントで“初回表示に必須のデータ”を取りに行って白画面を作らない
- SSR/Prefetch/Hydration を使う場合は「どこでデータが取得されるか」を docs に残す（暗黙にしない）

### SSR/Hydration は必須（Must）

本テンプレートでは SSR/Hydration を **必須（Must）** とします。

- 例外（CSR/SSG寄せ）を作る場合は、対象ページと理由を必ず docs に残す

概念整理・例外判断の基準： [SSR_HYDRATION.md](./SSR_HYDRATION.md)

### TanStack Query SSR/Hydration（実装テンプレ）

目的：
- 初期表示で必要なデータをサーバで埋めて返し、クライアント側の “最初の空白” を作らない
- ただし RSC で完結できる表示は Client 化せず、Hydration コストを払わない

基本手順（推奨）：
- Server（RSC）で **request ごとに QueryClient を作る**
- 必要な query を `prefetchQuery` し、`dehydrate` する
- Client 側は `HydrationBoundary` で受け取り、同じ queryKey の hooks が即座に参照できる状態にする

注意点（Must）：
- QueryClient を module-scope で共有しない（リクエスト間のデータ汚染）
- queryKey は必ず安定（パラメータ含む）にし、Server/Client で一致させる
- RSC から Route Handler を呼ばない（サーバ内で余計な HTTP hop）

追加（Must）：
- `prefetchQuery` は失敗しても throw しない（サーバで “欠けた状態” のままレンダリングされ、クライアントで再試行されうる）
- クリティカルな取得（404/500 を正しく返したい等）は `fetchQuery` を使い、失敗を明示的に扱う
- dehydratedState のシリアライズに注意（カスタム SSR での安易な stringify は XSS の原因になりうる）

運用（Should）：
- SSR では `staleTime` を 0 より上げることを検討（初期表示直後の不要な再フェッチを避ける）
- サーバ側のメモリ消費に注意（高トラフィック + QueryClient 多数生成）。必要なら request 終了後に `queryClient.clear()` を検討

設定の置き場所（推奨）：
- QueryClientProvider 等の “グローバル provider” は `src/app/providers`
- ページ単位の prefetch は `src/pages/<slice>` の合成点で行い、必要な Client コンポーネントへ渡す

運用メモ：
- mutation 後の整合は原則 `invalidateQueries`（正しさ優先）
- ただし Next のキャッシュ（tag/path）と二重管理しない方針を守る（詳細は `docs/01_architecture/CACHING_STRATEGY.md`）

---

## テスト時の方針（最小）

- Unit：純粋関数（`shared/lib` 等）
- Component：UIとユーザー操作（React Testing Library）
- Integration：API は MSW（OrvalのMSW生成も選択肢）
