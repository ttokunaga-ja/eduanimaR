# FSD Layers（運用ルール）

このドキュメントは、FSD（Feature-Sliced Design）における**レイヤーの責務**と、実装時に破りやすい**依存/配置ルール**を固定するための「契約」です。

- FSDの概要と判断基準： [FSD_OVERVIEW.md](./FSD_OVERVIEW.md)
- sliceの一覧（追加前に更新）： [SLICES_MAP.md](./SLICES_MAP.md)

---

## 1) 依存ルール（最重要）

### 1.1 レイヤー依存（単方向）
import は必ず上→下のみ：

`app` → `pages` → `widgets` → `features` → `entities` → `shared`

- ✅ 許可：`features/*` → `entities/*` / `shared/*`
- ❌ 禁止：`entities/*` → `features/*`

### 1.2 同一レイヤー内の Isolation
同一レイヤーの別sliceへ直接依存しません。

- ❌ 原則禁止：`features/a` → `features/b`
- 例外が必要なら（優先順）：
	1. 共通化できるものを `shared` へ移す
	2. 上位（`widgets` / `pages`）で合成する
	3. 設計自体（sliceの切り方）を見直す

---

## 2) Public API（`index.ts`）

各sliceはトップレベルに Public API を持ち、外部はそこからのみ import します。

- ✅ `import { UserCard } from '@/entities/user'`
- ❌ `import { UserCard } from '@/entities/user/ui/UserCard'`

Public API に公開するもの（目安）：
- `ui`：画面合成に必要なコンポーネント（例：`UserCard`）
- `model`：外部から利用される hook / actions（必要最小限）
- `api`：外部から叩く必要がある場合のみ（基本は `shared/api` に寄せる）

---

## 3) レイヤー別の責務（何を置くか）

### app（Application / ルート）
- **責務**：アプリ初期化、Provider、グローバル設定、エラー境界、ルーティングの殻
- **Next.js App Router 採用時**：`src/app` は App Router のディレクトリでもあるため、本テンプレではここを *appレイヤー* として扱います
- **置くもの**：
	- `layout.tsx`（Providers / global styles / metadata）
	- `providers/*`（QueryClientProvider、Theme、i18n 等）
	- `error.tsx` / `not-found.tsx`（必要な場合）
- **置かないもの**：画面固有の実装（ビジネスUIの本体）

### pages（画面の実体）
- **責務**：ルート（URL）に対応する画面を、widgets/features/entitiesで組み立てる
- **置くもの**：`ui/Page.tsx`（画面の合成）、ページ専用の薄い整形
- **置かないもの**：再利用前提の部品（再利用したいなら `widgets`/`features`/`entities`）

### widgets（独立したUIブロック）
- **責務**：複数feature/entityを合成する“塊”（例：ヘッダー、検索結果パネル）
- **置くもの**：レイアウトを含む UI ブロック、複合コンポーネント

### features（ユーザー価値の単位）
- **責務**：ユーザー操作 + ユースケース（例：ログイン、カート追加）
- **置くもの**：フォーム/操作UI、mutation、入力検証、成功/失敗の分岐
- **置かないもの**：アプリ全体の状態管理のハブ（features間依存を作りがち）

### entities（ビジネス実体）
- **責務**：ドメインオブジェクトの表現（表示・最小限の操作）
- **置くもの**：`UserCard` 等の表示、id→表示に必要な最小ロジック
- **置かないもの**：複数entity/featureをまたぐユースケース

### shared（共通基盤）
- **責務**：横断的に再利用される基盤（ビジネスルール禁止）
- **置くもの**：
	- `shared/ui`：UI primitives / wrappers
	- `shared/api`：OpenAPI生成物 + API共通設定
	- `shared/lib`：汎用関数
	- `shared/config`：環境変数、定数

---

## 4) Next.js での実装パターン（薄い adapter）

`src/app/**/page.tsx` はルーティングの“入口”で、原則 `src/pages/**/ui/Page` を import して描画するだけにします。

- 目的：FSDのページ実装を `pages` レイヤーへ集約し、ルーティング都合で構造が崩れるのを防ぐ

---

## 5) レビュー観点（チェックリスト）

- import が単方向（`app→...→shared`）になっている
- 同一レイヤー別sliceへの依存がない（特に `features→features`）
- deep import していない（Public API 経由）
- 置き場所が妥当（再利用するものを pages に閉じ込めていない）

---

## 6) ルールの強制（ツール）

人手レビューだけでは破綻しやすいため、境界ルールはツールで強制します。

- ESLint：`eslint-plugin-boundaries` で layers / slices の境界違反を検知
- import パス：`@/*` を `src/*` に割り当て、import を正規化（相対パス地獄を避ける）

注意：ここはプロジェクトの ESLint/tsconfig に依存するため、導入時に “実際のディレクトリ構造” に合わせて設定を確定させる。
