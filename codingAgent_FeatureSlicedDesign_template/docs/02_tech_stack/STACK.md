# 確定版：推奨技術スタック（2026年2月10日）

## Executive Summary (BLUF)
これまでの一連の分析に基づき、**Go製マイクロサービスバックエンド** と **FSD (Feature-Sliced Design)** を採用したフロントエンド開発における、**2026年時点での最適解となる技術スタック**を確定させました。

この構成は、**「型安全性の完全同期（Go⇔TS）」**、**「ゼロランタイムによる描画速度の最大化」**、そして**「大規模開発に耐えうる厳格なルール管理」**を同時に実現します。

---

## 1. 確定版：推奨技術スタック一覧

| カテゴリ | 推奨技術 | 役割と選定理由 |
| :--- | :--- | :--- |
| **Framework** | **Next.js (App Router)** | マイクロサービスを束ねる **BFF (Backend For Frontend)** として機能。Pigment CSS との親和性が高い。 |
| **Language** | **TypeScript** | 必須。Go の構造体と型定義を同期させるために使用。 |
| **UI System** | **MUI v6 + Pigment CSS** | **ゼロランタイム (Zero-runtime)** CSS。ビルド時に CSS を生成し、実行時の JS 負荷を減らして Core Web Vitals を改善する（※Pigment CSSは成熟途上のため運用上の注意点あり）。 |
| **State Mgt** | **TanStack Query v5/v6** | **サーバー状態管理**。Go サービスのデータをキャッシュ・同期する。FSD の `entities` / `features` 層で使用。 |
| **Client Gen** | **Orval**（or OpenAPI Generator） | **最重要**。Go (Echo) が出力する OpenAPI (Swagger) から TypeScript の型と fetch 関数を自動生成する。手書きの型定義を禁止し、齟齬バグを根絶する。 |
| **Validation** | **Zod** | スキーマバリデーション。フォーム入力値のチェックに使用。React Hook Form と連携。 |
| **Forms** | **React Hook Form** | 非制御コンポーネントベースで高速。MUI と統合して `features` 層に配置。 |
| **Testing** | **Vitest + Playwright** | 高速なユニット/コンポーネントテストと、E2E テストを両立する。 |
| **Linter** | **ESLint + `eslint-plugin-boundaries`** | FSD の **階層ルールを強制**する守護神。違反をエディタ/CI で検知し、人手レビュー依存を減らす。 |
| **Bundler** | **Turbopack**（Next.js 標準） | 高速な HMR（Hot Module Replacement）を実現。 |
| **Runtime** | **Node.js (Docker)** | `alpine` ベースの軽量イメージ。Next.js の `standalone` モードで運用。 |

---

## 1.1 最新版（取得日付き：SSOT）

このテンプレは特定プロジェクトの依存を同梱しないため、最新版は外部ソースから都度取得し、ここをSSOTとして更新します。

取得元：
- npm（dist-tag: latest）：`npm view <package> version`
- Node.js（公式）：`curl -fsSL https://nodejs.org/dist/index.json`

最新版（2026-02-11 に取得）：

| Tech | Package | Latest |
| --- | --- | --- |
| Next.js | `next` | `16.1.6` |
| React | `react` / `react-dom` | `19.2.4` / `19.2.4` |
| TypeScript | `typescript` | `5.9.3` |
| MUI | `@mui/material` | `7.3.7` |
| Pigment | `@pigment-css/react` | `0.0.30` |
| TanStack Query | `@tanstack/react-query` | `5.90.20` |
| Orval | `orval` | `8.2.0` |
| Zod | `zod` | `4.3.6` |
| React Hook Form | `react-hook-form` | `7.71.1` |
| Vitest | `vitest` | `4.0.18` |
| Playwright | `@playwright/test` | `1.58.2` |
| ESLint | `eslint` | `10.0.0` |
| Boundaries | `eslint-plugin-boundaries` | `5.4.0` |

Node（公式 index.json、2026-02-11 に取得）：
- latest LTS：`v24.13.1`（Krypton）
- latest Current：`v25.6.1`

---

## 2. アーキテクチャ構成図（FSD × Microservices）

```mermaid
graph TD
    subgraph "Dev Environment (Code Generation)"
        GoStructs[Go Structs] -->|Swag/OAPI-Codegen| OpenAPI[OpenAPI Spec (JSON/YAML)]
        OpenAPI -->|Orval| GenHooks[Generated React Hooks]
    end

    subgraph "Frontend (Next.js + FSD)"
        direction TB
        Page[Pages Layer] --> Widget[Widgets Layer]
        Widget --> Feature[Features Layer]
        Feature --> Entity[Entities Layer]
        Entity --> Shared[Shared Layer]
        
        Shared -->|Uses| GenHooks
        Shared -->|Uses| MUI[MUI + Pigment CSS]
    end

    subgraph "Backend (Go Microservices)"
        NextBFF[Next.js Server (BFF)] -->|HTTP/JSON (w/ JWT)| GoGateway[Go API Gateway Service\n(Echo v5)]
        GoGateway --> ServiceA[User Service (Echo)]
        GoGateway --> ServiceB[Search Service (Echo)]
        
        ServiceA --> DB[(PostgreSQL)]
        ServiceB --> ES[(Elasticsearch)]
    end

    GenHooks -.->|Fetch Data| NextBFF
```

---

## 3. 開発範囲（2段階ゲートウェイ：Next.js BFF × Go API Gateway）

本テンプレートは、以下の **2段階ゲートウェイ構成** を前提に「どこまで作るか（開発範囲）」を明示します。

1. **Next.js（BFF）**：UI のためのゲートウェイ（フロントエンド層 / Server Side）
2. **Go API Gateway**：システム全体のゲートウェイ（バックエンド層）

### Next.js（BFF）の開発範囲
- App Router（RSC / Route Handlers）を使い、**画面表示に必要なデータの整形・集約**を行う
- Cookie/Session 等の **ブラウザ向け状態** を扱い、必要に応じて JWT を取り出して Go Gateway に中継する
- **UI 最適化**（ページ単位キャッシュや、複数 API 結果の合成、表示用フォーマット）に集中する
- FSD に従い、`pages` / `widgets` / `features` / `entities` / `shared` の責務を守る

### Go API Gateway（バックエンド）の開発範囲
- **共通処理の集約**：認証（JWT 署名検証）、認可（RBAC）、レート制限、監査ログ、トレーシング等
- **ルーティング**：適切なマイクロサービスへ転送（パス書き換えやバージョニングを含む）
- **プロトコル変換**：外向きは HTTP/JSON、内向きは gRPC（または HTTP）
- **内部隠蔽**：ブラウザ/Next.js からマイクロサービスを直接叩かせず、入口を一本化する

### 各マイクロサービス（Go/Echo 等）の開発範囲
- **ビジネスロジックの実装**（ドメインルール、整合性、永続化）
- DB（PostgreSQL）や外部基盤（Elasticsearch 等）へのアクセス
- サービス単位で責務を閉じ、他サービスとの通信は原則 gRPC/HTTP（内部）で行う

### 契約（API スキーマ）の開発範囲
- Go 側（Gateway および各サービス）は OpenAPI を出力/保守する
- フロントエンドは **Orval による生成物** を唯一の API クライアントとして使用し、手書き型定義を禁止する

### 明確に「やらない」こと（境界の固定）
- Next.js がマイクロサービスへ **直接** 接続する（内部構造の露出を招く）
- Go API Gateway に **ビジネスロジック** を書く（Gateway は土管 + 守護神に徹する）
- フロントエンドで API 型定義や fetch/axios を **手書き** する（生成に統一）

---

## 4. FSDディレクトリ構造の具体例

```text
src/
├── app/                  # Next.js App Router（routing / providers の殻）
│   ├── layout.tsx        # Providers / Pigment CSS の設定
│   └── (routes)/...      # 原則: src/pages をimportして表示するだけ（薄いadapter）
│
├── pages/                # FSD: Pages Layer (ページ単位の組み立て)
│   └── user-profile/
│       └── ui/
│           └── Page.tsx
│
├── widgets/              # FSD: Widgets Layer (大きなUIブロック)
│   └── header/
│
├── features/             # FSD: Features Layer (機能・ユースケース)
│   └── auth/
│       ├── login-form/   # RHF + Zod + MUI
│       └── model/        # 状態管理ロジック
│
├── entities/             # FSD: Entities Layer (ビジネス実体)
│   └── user/
│       ├── ui/           # UserCard (MUI Component)
│       └── model/        # userStore (Zustand if needed)
│
└── shared/               # FSD: Shared Layer (共通部品・設定)
    ├── api/              # Orvalで自動生成されたコード (user.gen.ts etc)
    ├── ui/               # MUIのラッパー (Button, Input)
    ├── config/
    │   └── theme.ts      # Pigment CSS Theme設定
    └── lib/              # Utils
```

---

## 5. 成功のための3つの鉄則

### ① 型定義は「書かない」、生成する
- **Go エンジニア**は、Echo のハンドラーにコメント（Swag）を書く、またはコードから OpenAPI 仕様を出力する責任を持つ。
- **フロントエンドエンジニア**は、`npm run api:generate` コマンド一つで、Go 側の変更（例：User 構造体に `age` が増えた）を TypeScript の型として即座に取り込む。
- これにより、**バックエンドとフロントエンドの認識齟齬**によるバグを抑止する。

### ② CSSは「実行時」に計算させない
- MUI v5（emotion）までは、ブラウザで JS が動いてスタイルを計算していた。
- Pigment CSS を使う本構成では、**動的なスタイル変更（`sx` prop など）の使用を `shared/ui` 内のコンポーネントに限定**し、可能な限りビルド時に CSS を確定させる。

補足（重要）：Pigment CSS は仕様/実装の更新が続く可能性があるため、導入時は [MUI_PIGMENT.md](./MUI_PIGMENT.md) の「DO/DON'T」とアップグレード時の確認観点を必ず守る。

### ③ FSDの境界線（Boundaries）を絶対守る
- 「便利だから」といって、`entities` から `features` を import してはいけない。
- `eslint-plugin-boundaries` を導入し、**CI（自動テスト）で違反があればマージできない**ように設定する。

---

## 結論
提示されたバックエンド（Go, Echo, Elasticsearch, PostgreSQL）に対し、このフロントエンドスタック（Next.js, FSD, MUI+Pigment, Orval）は、**型安全・生成駆動・境界強制**を同時に満たす有力な組み合わせです。
