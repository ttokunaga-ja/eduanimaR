# Feature-Sliced Design (FSD) Overview（運用版）

Last-updated: 2026-02-16

このドキュメントは、FSD（Feature-Sliced Design）の核心・2026年時点の実践パターン・実装例をまとめ、AI/人間が同じ判断基準で設計できるようにするための「共通認識」です。

- レイヤーの責務定義は [FSD_LAYERS.md](./FSD_LAYERS.md)
- どの機能がどの slice に属するかは [SLICES_MAP.md](./SLICES_MAP.md)
- 本テンプレの確定版技術スタックは [STACK.md](../02_tech_stack/STACK.md)

---

## Executive Summary（結論）
FSDは、コードを「技術的な役割（Components/Hooks）」ではなく、**「ビジネス価値（Features/Entities）」**を中心に分割・階層化するアーキテクチャです。

最大の特徴は、**上層は下層のみを import できる（単方向依存）**という厳格なルールにあり、機能間の結合度を下げ（Low Coupling）、凝集度を高める（High Cohesion）ことで、規模が大きくなってもコードベースのスパゲッティ化を防ぎます。

FSDの重要コンセプト（公式の要点）：
- **Public API**：各モジュールはトップレベルに公開面（`index.ts` 等）を定義し、外部からの参照点を固定する
- **Isolation**：同一レイヤーの別sliceに直接依存しない（必要なら上位で合成する）

---

## 情報階層の原則（UI設計の指針）

eduanimaRでは、以下の情報階層を厳守します：

### 1. 根拠（Evidence）→ 2. 要点 → 3. 次の行動

**根拠提示（Evidence-forward）**:
- 回答には必ず参照元資料がクリッカブルに添付される
- 資料名、ページ番号、セクション名を明示
- 抜粋は引用として分かる体裁を維持

**トーン&マナー（UI表現の原則）**:
- 落ち着いて、正確で、学習者に敬意のある表現を維持
- 断定よりも根拠・前提を示す
- 複雑さを増やさず、次の一歩を短く提示する

**参照元SSOT**:
- `../../eduanimaRHandbook/04_product/VISUAL_IDENTITY.md` (情報階層)
- `../../eduanimaRHandbook/04_product/BRAND_GUIDELINES.md` (トーン&マナー)

---

## eduanimaRにおけるFSDの適用

本プロジェクトでは、以下のバックエンドサービスと連携します：

- **Professor（Go）**: 外向きAPI（OpenAPI）、DB/GCS管理、最終回答生成
- **Librarian（Python）**: 推論ループ（LangGraph）、検索戦略立案

フロントエンドの責務：
- Professor APIとの統合（Orval生成クライアント使用）
- SSEによるリアルタイム推論状態の表示
- Chrome拡張機能による自動アップロード

**FSDの適用により、バックエンドとの境界を明確にし、API契約（OpenAPI）を介した疎結合を実現します。**

### Monorepo構成とコード共有戦略

本プロジェクトは**WebアプリとChrome拡張機能を同時提供**するため、Monorepo構成を前提とします。

#### ディレクトリ構成（想定）

```
apps/
  ├── web/                      # Next.js（App Router） - Webアプリ
  │   ├── src/
  │   │   ├── app/              # Next.js App Router（routing/providers）
  │   │   ├── pages/            # FSD Pages（画面実体）
  │   │   ├── widgets/          # 合成Widget
  │   │   ├── features/         # ユーザー価値機能
  │   │   ├── entities/         # ビジネス実体
  │   │   └── shared/           # Web固有の共通部品
  │   └── package.json
  └── extension/                # Plasmo Framework - Chrome拡張機能
      ├── src/
      │   ├── contents/         # Content Scripts
      │   ├── background/       # Background/Service Worker
      │   ├── sidepanel/        # Sidepanel（質問UI）
      │   ├── popup/            # Popup（設定UI）
      │   ├── features/         # 拡張機能固有のFeatures
      │   ├── entities/         # 共有Entity（packages/から参照）
      │   └── shared/           # 拡張機能固有の共通部品
      └── package.json
packages/
  ├── shared-api/               # Orval生成クライアント（共通）
  │   ├── src/
  │   │   ├── generated/        # 自動生成
  │   │   ├── client.ts         # baseURL/認証設定
  │   │   └── index.ts          # Public API
  │   └── package.json
  ├── shared-ui/                # FSD shared/ui（共通コンポーネント）
  │   ├── src/
  │   │   ├── button/
  │   │   ├── evidence-card/    # 根拠資料カード（QAチャット用）
  │   │   └── index.ts
  │   └── package.json
  └── shared-types/             # 共通型定義
      ├── src/
      │   └── index.ts
      └── package.json
```

#### 共有方針（FSD層別）

| FSD層 | 共有方針 | 配置 | 例 |
|------|---------|------|-----|
| **Shared** | ✅ **積極的に共有** | `packages/shared-*` | ボタン、カード、型定義、API Client |
| **Entities** | ✅ **ビジネスロジックが同一なら共有** | `packages/entities` or 各アプリに配置 | `entities/user`（表示/状態） |
| **Features** | △ **ケースバイケース** | 基本は各アプリ固有、共通化できるなら`packages/features` | `features/qa-chat`（Web/拡張で共通化可能） |
| **Widgets** | ❌ **各アプリ固有** | 各アプリ内 | Web: `widgets/file-tree`、拡張: `widgets/upload-status` |
| **Pages** | ❌ **必ず各アプリ固有** | 各アプリ内 | Web: `pages/home`、拡張: `sidepanel/qa` |

#### 実装ガイドライン

1. **API通信の統一**:
   - Orval生成クライアント（`packages/shared-api`）を両アプリで共有
   - Web: Server Components/Route Handler、拡張: Background/Service Worker から呼び出し

2. **UI資産の共有**:
   - MUI + Pigment CSS を両アプリで使用
   - 共通コンポーネント（`EvidenceCard`, `LoadingSpinner` 等）は`packages/shared-ui`へ配置
   - 拡張機能ではShadow DOM隔離戦略を適用

3. **FSD境界の維持**:
   - `packages/*`も FSD Public API（`index.ts`）を徹底
   - 各アプリ内の`features`/`entities`は、まず各アプリに配置し、共通化が明確になったら`packages/*`へ移動

4. **依存方向**:
   - `apps/*` → `packages/*` の一方向依存のみ許可
   - `packages/*` 間の依存は最小限に（`shared-types` ← `shared-api` ← `shared-ui`）

**参照**: 
- [`../../eduanimaRHandbook/02_strategy/TECHNICAL_STRATEGY.md`](../../eduanimaRHandbook/02_strategy/TECHNICAL_STRATEGY.md) L112-150
- FSD公式ドキュメント: https://feature-sliced.design/

---

## 1. FSDの構造：3つの階層レベル

### Level 1: Layers（レイヤー）
最上位の分類です。**上→下への依存のみ許可**します。

| 階層（上→下） | 名称 | 役割 |
| :--- | :--- | :--- |
| 1 | App | アプリ初期化、Provider、グローバル設定（ルーティング含む） |
| 2 | Pages | ルーティングに対応する画面の組み立て（配置・合成） |
| 3 | Widgets | 独立した UI ブロック（複数 Feature の合成地点） |
| 4 | Features | ユーザー価値のある機能（UI + ユースケース） |
| 5 | Entities | ビジネス実体（モデル/表示/CRUD の基礎） |
| 6 | Shared | 再利用可能な共通部品（ビジネスロジック禁止） |

補足：本テンプレートでは、複雑フローの専用レイヤー（例：`processes`）は **採用しません**。必要な合成は `pages` / `widgets` / `features` のいずれかで表現します。

### Level 2: Slices（スライス）
レイヤー内を、ビジネス領域ごとに分割します。

- 例：`entities/user`、`features/auth-by-token`、`pages/home`
- ルール：**同一レイヤーの slice 同士は原則 import しない**
  - 例：`features/A` → `features/B` の直接 import は避ける
  - クロスが必要なら：
    - 共通化できる部分を `shared` に落とす、または
    - 上位の `widgets` / `pages` で合成する

スライスの追加/変更は、必ず先に [SLICES_MAP.md](./SLICES_MAP.md) を更新します。

### Level 3: Segments（セグメント）
スライス内を技術的な役割で分けます。

- `ui`：コンポーネント
- `model`：状態・ユースケース（フォーム状態、UI state、ドメイン操作）
- `api`：通信（本テンプレでは原則 Orval 生成物を利用）
- `lib`：ユーティリティ（スライス内限定の helper など）

補足：Segmentsは「必要になったら増やす」が基本です。最初から `ui/model/api/lib` を全sliceに作らない（空ディレクトリを量産しない）。

---

## 2. 2026年時点のベストプラクティス

### 2.1 Next.js（App Router）との共存パターン
Next.js の App Router は `app/`（または `src/app/`）にルーティングとレイアウトの責務が集まります。
本テンプレでは **`src/app` を「routing / providers の殻」として扱い、画面の実体は `src/pages` から import する**運用を基本とします。

例（方針のイメージ）：

```text
src/
├── app/                         # Next.js App Router（routing/providesのみ）
│   ├── layout.tsx               # Providers / global styles / metadata
│   └── (shop)/products/page.tsx # src/pages を import して描画するだけ
└── pages/                       # FSD Pages（画面実体）
  └── products/
    └── ui/Page.tsx
```

Server Components（RSC）の扱い：
- まず Server Component として設計し、インタラクションが必要な部分だけを Client Component（`"use client"`）に切り出します
- Client 化の波及でレイヤー境界が崩れないよう、合成点は `pages` / `widgets` に寄せます

RSC とデータ取得（本テンプレのポリシー）：
- **RSC（Server）**：必要なら「生成クライアント（非Hook）」で取得して表示用に整形する（BFF責務）。Client向けに生データをばらまかない。
- **Client**：サーバー状態は TanStack Query（生成Hooks）へ寄せ、キャッシュ/無効化/再取得を一元化する。

### 2.2 Public API（`index.ts`）の徹底
各 slice は Public API（`index.ts`）を持ち、外部に公開するモジュールを制限します（カプセル化）。

- Good：`import { UserCard } from '@/entities/user'`
- Bad：`import { UserCard } from '@/entities/user/ui/UserCard'`

Public API の目的：
- リファクタ容易性（内部構造変更の影響を最小化）
- import ルールの単純化（ESLint で強制しやすい）

運用ルール（AI/人間共通）：
- `features/*/ui/*` の深い import を見かけたら、まず `features/<slice>/index.ts` に公開し直す
- 何を公開するか迷ったら「上位レイヤーが合成に必要な最小限のみ」を公開する

### 2.3 ルールは「ツールで強制」する
FSDはルールが多く、人間の記憶に頼る運用は破綻しやすいです。本テンプレートでは、まず以下を前提にします。

- **ESLint + `eslint-plugin-boundaries`**：レイヤー/スライス境界の違反を CI/エディタで検知

補足（採用はプロジェクト方針に依存）：
- FSD専用 Linter（例：Steiger 等）を追加して、依存違反の可視化を強化する選択肢もあります

関連（運用オプション）：
- TanStack Query は ESLint plugin を併用すると、queryKey / deps の事故を減らせます（採用可否はプロジェクトで決める）

---

## 2.4 よくある迷いどころ（置き場所の規約）

クロスカット（横断）関心事の置き場所：
- **認証/セッション**：`entities/session`（表示/状態） + `features/auth-*`（ログイン/ログアウト等のユースケース）
- **通知（toast/snackbar）**：`shared/ui` に土台、発火は各 feature から。グローバル制御が必要なら `app/providers` に集約
- **エラーハンドリング**：UIは `app`（error boundary / route error）に置き、APIエラーの分類は `shared/api` で統一
- **i18n**：辞書/フォーマットは `shared/lib/i18n`、画面文言は原則 `pages`/`widgets`/`features` の `ui` 側に置く
- **UIに表示する文字列はすべて翻訳キー（変数）として管理**し、表示内容は各言語ごとの JSON ファイル（例：`public/locales/{lang}/common.json` または `src/locales/{lang}.json`）から読み出すことを**必須**とする（詳細は `../03_integration/I18N_LOCALE.md` を参照）

---

## 3. 実装事例：ECサイトの「カートに追加」機能

### 分割の考え方
- Shared：ビジネスロジックを含まない再利用（UI Kit、API生成物、utils）
- Entities：ビジネス実体（Product/Cart）
- Features：ユーザー価値（AddToCart）
- Widgets：複数 Feature/Entity の合成（商品一覧）
- Pages：画面としての組み立て（ルーティング単位）

### 配置例（概念）
1. **Shared**：`shared/ui/button`（汎用ボタン）
2. **Entities（product）**：`entities/product`（モデル、`ProductCard`）
3. **Entities（cart）**：`entities/cart`（バッジなど表示、基礎操作）
4. **Features（add-to-cart）**：ボタン押下 → API 呼び出し → キャッシュ更新等のユースケース
5. **Widgets（product-list）**：`ProductCard` と `AddToCart` を合成して一覧表示
6. **Pages（shop）**：`product-list` widget を配置して画面を構成

ポイント：
- 「A feature が別の feature を直接使いたい」状況は設計臭が強い（合成点を上位へ移す）
- データ取得は、可能な限り生成クライアント（Orval） + Query（TanStack Query）に寄せる

---

## 4. 導入の判断基準

### FSDが適しているケース
- 開発期間が半年以上、または中〜大規模
- 開発メンバーが3人以上（入れ替わりがある）
- 「置き場所の議論」「依存のねじれ」「レビュー負荷」を減らしたい

### FSDが適さないケース
- LPや数ページの小規模サイト
- 使い捨てプロトタイプ
- 初期学習コストを吸収できない状況（ただし、ルールを最小限にして段階導入する選択肢はある）
