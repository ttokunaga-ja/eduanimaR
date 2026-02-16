---
Title: Tech Stack
Description: eduanimaRプロジェクトの技術スタック一覧と選定理由
Owner: @ttokunaga-ja
Status: Published
Last-updated: 2026-02-16
Tags: frontend, eduanimaR, tech-stack, backend, api
---

# 確定版：推奨技術スタック（2026年2月10日）

Last-updated: 2026-02-16

## サービスミッション（North Star）

**Mission**: 学習者が、配布資料や講義情報の中から「今見るべき場所」と「次に取るべき行動」を素早く特定できるようにし、理解と継続を支援する

**Vision**: 必要な情報が、必要なときに、必要な文脈で見つかり、学習者が自律的に学習を設計できる状態を当たり前にする

**North Star Metric**: 資料から根拠箇所に到達するまでの時間短縮

**参照**: [`../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md`](../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md)

### 独自価値提案（Unique Value Proposition）

> **「あなたのLMS資料を、あなた専用の生きた知識ベースに変える司書と教授」**

**Vision Reasoning（画像・数式の意味理解）**:
- 図やグラフ、数式を「意味」として理解（単なるテキスト抽出ではない）

**LangGraph Agent（自動再試行検索パターン）**:
- 検索戦略を自律的に立案・修正
- 高い資料発見率を実現

**Go/Python ハイブリッド**:
- 堅牢なデータ管理（Go）+ 高度なAI推論（Python）の組み合わせ

**参照**: [`../../eduanimaRHandbook/02_strategy/LEAN_CANVAS.md`](../../eduanimaRHandbook/02_strategy/LEAN_CANVAS.md)

### 提供価値（学習支援特化）

eduanimaRは「資料の着眼点を示し、原典への回帰を促す」学習支援ツールです：

- **探索支援**: 資料のどこに何が書いてあるかを素早く特定
- **理解支援**: 重要箇所を示し、学習者の理解を促進
- **学習計画**: 次に何を学ぶべきかを明確化

**原則**:
- 評価・試験での不正な優位を得る目的での利用は想定しない
- 学習者の自律的な学習を支援する

**参照**: [`../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md`](../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md)

## バックエンドスタック概要

フロントエンドが依存するバックエンド（Professor/Librarian）のスタック:

| 項目 | バージョン | 備考 |
|------|-----------|------|
| **Go** | 1.25.7 | Professor（データ守護者/ゲートウェイ） |
| **Python** | 3.12+ | Librarian（推論エンジン） |
| **PostgreSQL** | 18.1 | pgvector 0.8.1含む、Professor専有 |
| **Echo** | v5.0.1 | Professor HTTP/JSON + SSE API |
| **Litestar** | - | Librarian HTTP/JSON API |
| **Google Cloud Run** | - | Professor/Librarian実行基盤 |
| **Google Cloud Storage** | - | 講義資料ストレージ、Professor専有 |

## API契約のバージョン管理

- **OpenAPI SSOT**: `eduanimaR_Professor/docs/openapi.yaml`
- **フロントエンド生成**: Orvalで型・クライアント自動生成
- **バージョニング**: `/v1/`, `/v2/` 形式
- **Breaking Changes**: Professor側で明記、フロントエンド側で移行計画

## バックエンド統合とProfessor/Librarian責務境界

| 役割 | 技術 | 備考 |
| --- | --- | --- |
| 外向きAPI | Professor（Go） | OpenAPI仕様提供、HTTP/JSON + SSE |
| 推論エンジン | Librarian（Python） | LangGraph + Gemini 3 Flash |
| 内部通信 | **gRPC（双方向ストリーミング）** | Professor ↔ Librarian、契約: `proto/librarian/v1/librarian.proto` |
| ストリーミング | SSE (Server-Sent Events) | `/v1/qa/ask` |
| API生成 | Orval | OpenAPI → TypeScript |

### Professor/Librarian責務境界の徹底明記

#### Professor（Go）の責務
- **データ守護者（唯一の権限者）**: DB/GCS/Kafka直接アクセス権限を持つ
- **Phase 2（大戦略）**: タスク分割・停止条件決定
- **Phase 3（物理実行）**: 
  - ハイブリッド検索（RRF統合）
  - 動的k値設定
  - 権限強制
- **Phase 4（合成）**: Gemini 3 Proで最終回答生成
- **外向きAPI提供**: HTTP/JSON + SSEでフロントエンドと通信

#### Librarian（Python）の責務
- **Phase 3（小戦略）**: LangGraphによる推論ループ（最大5回推奨）
- **ステートレス推論サービス**: 会話履歴・キャッシュなし
- **DB直接アクセス禁止**: Professor経由でのみ検索実行
- **通信**: **gRPC（双方向ストリーミング）** でProfessorと通信、契約: `proto/librarian/v1/librarian.proto`

#### Frontend（Next.js）の責務
- **ProfessorのHTTP/JSON+SSEのみ**: Librarian直接通信禁止
- **選定エビデンス表示**: Librarian推論ループが選定した根拠箇所をUI表示
- **会話履歴管理**: Librarianがステートレスのため、クライアント側で保持

### Professor OpenAPI契約の詳細（SSEストリーミング・エビデンス表示）

#### SSEイベントタイプと処理要件

Professor の `/v1/qa/stream` エンドポイントは、以下のSSEイベントをリアルタイム配信します：

| イベントタイプ | 内容 | フロントエンド処理 |
|:---|:---|:---|
| `thinking` | Phase 2実行中（タスク分割・停止条件生成） | プログレス表示「AI Agentが検索方針を決定しています」 |
| `searching` | Librarian推論ループ実行中（最大5回） | プログレスバー更新（例：「2/5回目の検索」） |
| `evidence` | 選定エビデンス提示 | エビデンスカード表示（クリッカブルURL、why_relevant、snippets） |
| `answer` | 最終回答生成中（Gemini 3 Pro） | リアルタイムにテキスト追加表示 |
| `done` | 完了通知 | SSE接続を閉じる |
| `error` | エラー通知 | エラートースト表示 |

#### エビデンス表示の必須要素

Professor OpenAPI契約に基づく、エビデンスカードの必須表示要素：

- **クリッカブルpath/url**: GCS署名付きURLで原典にアクセス可能
- **ページ番号（page）**: 該当箇所のページ番号（例：「p.3」）
- **why_relevant**: なぜこの箇所が選ばれたかの説明文
- **snippets**: 資料からの抜粋（Markdown形式）
- **heading**: 該当セクションの見出し

**実装要件**:
- エビデンスカードは「主役」として画面上部に配置（情報階層に基づく）
- クリック時に原典（PDF/GCSリンク）へ遷移
- why_relevantを明示し、学習者が「なぜ」を理解できるようにする

**参照**: [`../../eduanimaR_Professor/docs/03_integration/ERROR_CODES.md`](../../eduanimaR_Professor/docs/03_integration/ERROR_CODES.md)、[`../../eduanimaRHandbook/04_product/VISUAL_IDENTITY.md`](../../eduanimaRHandbook/04_product/VISUAL_IDENTITY.md)

### Gemini モデル役割分担
- **Gemini 3 Flash**: 
  - Professor: Phase 2（大戦略）、インジェスト（PDF→Markdown）
  - Librarian: Phase 3（小戦略）、Librarian推論ループ
- **Gemini 3 Pro**: 
  - Professor: Phase 4（最終回答生成）

### 検索戦略の詳細（Phase 3物理実行）

Professor が実行するハイブリッド検索戦略:

| 検索手法 | 役割 | 利点 |
|---------|------|------|
| **全文検索（基盤）** | PostgreSQL全文検索 | 固有名詞・専門用語に強い |
| **pgvector併用** | ベクトル類似検索 | 同義語・言い換え対応 |
| **ハイブリッドRRF統合** | Reciprocal Rank Fusion (k=60) | 両手法の長所を統合 |
| **動的k値設定** | 件数に応じた調整 | N < 1,000: k=5 / N ≥ 100,000: k=20 |

参照：
- [`../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md`](../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md)
- [`../../eduanimaR_Professor/docs/README.md`](../../eduanimaR_Professor/docs/README.md)

## Professor API通信（契約駆動開発）

### フロントエンドの通信ルール
- **Professor API**: フロントエンドはProfessorのOpenAPI(HTTP/JSON + SSE)経由でバックエンドと通信
- **Librarian呼び出し禁止**: LibrarianはProfessor経由でのみ呼び出され、フロントエンドから直接呼び出さない
- **契約駆動開発**: OpenAPIからの型/クライアント生成を必須化(手書きの型定義・fetch関数を禁止)

**参照元SSOT**:
- `../../eduanimaR_Professor/docs/02_tech_stack/STACK.md`
- `../../eduanimaR_Professor/docs/02_tech_stack/TS_GUIDE.md`
- `../../eduanimaR_Professor/proto/librarian/v1/librarian.proto` (コメント)

## Phase別の技術スタック差異

### Phase 1（ローカル開発）
- 認証: スキップ（固定dev-user）
- API接続: ローカルProfessor（`http://localhost:8080`）

### Phase 2（本番環境）
- 認証: SSO（NextAuth.js/Auth.js + Professor OAuth/OIDC）
- API接続: Cloud Run（`https://professor.example.com`）
- **重要制約**:
  - **新規ユーザー登録はChrome拡張機能でのみ許可**
  - **Web版は既存ユーザーのログイン専用**
  - **未登録ユーザーは拡張機能ダウンロードページへ誘導**

### Phase 3（Librarian推論ループ）
- SSE: リアルタイム回答ストリーミング
- Librarian推論ループ（フロントエンドからは不可視）
- Professor経由でのみLibrarianと連携

### Phase 4（学習計画）
- カレンダーUI、進捗管理機能

## eduanimaR 固有の前提（2026-02-15更新）

本プロジェクトは、**大学LMS資料の自動収集・検索・学習支援**を提供する以下の構成です：

| コンポーネント | 役割 | 技術スタック |
|:---|:---|:---|
| **Frontend** | Chrome拡張機能 + Webアプリ | Next.js 15 (App Router) + FSD + MUI v6 + Pigment CSS |
| **Professor（Go）** | 外向きAPI（HTTP/JSON + SSE）、DB/GCS/Kafka管理、最終回答生成 | Go 1.25.7, Echo v5, PostgreSQL 18.1 + pgvector 0.8.1, Google Cloud Run |
| **Librarian（Python）** | LangGraph Agent による検索戦略立案 | Python 3.12+, Litestar, LangGraph, Gemini 3 Flash |

### データフローと責務境界
1. **Frontend → Professor**: OpenAPI（HTTP/JSON）でリクエスト送信
2. **Professor ↔ Librarian**: **gRPC（双方向ストリーミング）** で検索戦略の協調
   - Professor: Phase 3物理実行（ハイブリッド検索(RRF統合)、動的k値設定）
   - Librarian: Phase 3小戦略（Librarian推論ループ、最大5回推奨）
   - 契約: `eduanimaR_Professor/proto/librarian/v1/librarian.proto`
3. **Professor → Frontend**: SSEでリアルタイム回答配信（選定エビデンス含む）
4. **Professor**: Kafka経由でOCR/Embeddingのバッチ処理（DB/GCS直接アクセス）

### 認証（Phase 2以降）
- SSO（OAuth 2.0 / OpenID Connect）
- 対応プロバイダ: Google / Meta / Microsoft / LINE
- Phase 1（ローカル開発）: 認証スキップ（固定dev-user）
- **Phase 2の重要制約**:
  - **新規ユーザー登録はChrome拡張機能でのみ許可**
  - **Web版は既存ユーザーのログイン専用**
  - **未登録ユーザーは拡張機能ダウンロードページへ誘導**

### サービスコンセプト（eduanimaRHandbook より）
- **Mission**: 学習者が、配布資料や講義情報の中から「今見るべき場所」と「次に取るべき行動」を素早く特定できるようにし、理解と継続を支援する
- **North Star Metric**: 資料から根拠箇所に到達するまでの時間短縮
- **主要ペルソナ**: 忙しい学部生（複数科目、資料が散在、探す時間が負担）
- **提供価値**: 資料の「着眼点」を示し、原典への回帰を促す

参照: [`../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md`](../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md)

### 提供形態とChrome拡張/Web役割分離（Phase 2以降）

1. **Chrome拡張機能（メインチャネル）**
   - **Phase 2: ユーザー登録可能な唯一の手段**
   - Moodle資料の完全自動収集（最重要機能）
   - LMS上でのSSO認証・ユーザー登録
   - 履修科目の自動同期
   - その場で質問・参照

2. **Webアプリケーション（補助チャネル）**
   - **既存ユーザーログイン専用（新規登録不可）**
   - 大画面でのチャット・履歴閲覧
   - 拡張機能で登録したユーザーの再ログイン専用
   - **新規登録・科目登録・ファイルアップロードは無効化**
   - **未登録ユーザー誘導**: Chrome Web Store/GitHub/導入ガイドへの誘導

### バックエンド構成と責務分担
- **Professor（Go）**: データ守護者（DB/GCS/Kafka直接アクセス唯一の権限者）、外向きAPI（HTTP/JSON + SSE）、最終回答生成
- **Librarian（Python）**: 推論特化（LangGraph Agent）、ステートレス、Professor経由でのみ検索実行、DB直接アクセス禁止
- **Frontend**: Professorの外部APIのみを呼ぶ（Librarianへの直接通信禁止）

### バックエンド技術スタック（参考）
| コンポーネント | 技術 |
|--------------|------|
| Professor | Go 1.25.7, Echo v5, PostgreSQL 18.1 + pgvector 0.8.1, Gemini 3 Flash/Pro |
| Librarian | Python 3.12+, Litestar, LangGraph, Gemini 3 Flash |
| 通信 | Frontend ↔ Professor: HTTP/JSON + SSE, Professor ↔ Librarian: **HTTP/JSON** |

### 認証方式
- Phase 1: dev-user固定（ローカル開発のみ）
- Phase 2以降: SSO（Google / Meta / Microsoft / LINE）
- **Web版からの新規登録禁止**: 拡張機能でSSO登録したユーザーのみログイン可能

---

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

### バックエンド技術スタック概要

| サービス | 役割 | 技術スタック |
|:---|:---|:---|
| **Professor** | データ所有者、DB/GCS/Kafka 直接アクセス、検索の物理実行、最終回答生成 | Go 1.25.7, Echo v5.0.1, PostgreSQL 18.1 + pgvector 0.8.1, Google Cloud Run |
| **Librarian** | 推論特化、検索戦略立案（Professor 経由でのみ検索実行） | Python 3.12+, Litestar, LangGraph, Gemini 3 Flash |

### Professor ↔ Librarian 通信
- **プロトコル**: **gRPC（双方向ストリーミング）**
- **契約**: `eduanimaR_Professor/proto/librarian/v1/librarian.proto`
- **Librarianの特性**: ステートレス推論サービス（会話履歴・キャッシュなし）
- **技術的理由**: Phase 3検索ループにおける複数ターン双方向通信に最適

### 責務分担の明確化（Professor/Librarian境界）

#### Professor（Go）の責務
- **データ守護者（唯一の権限者）**: DB/GCS/Kafka への直接アクセス権限を持つ
- **Phase 2（大戦略）**: タスク分割・停止条件決定
- **Phase 3（物理実行）**: 
  - ハイブリッド検索（RRF統合、k=60）
  - 動的k値設定（N < 1,000: k=5, N ≥ 100,000: k=20）
  - 権限強制（ユーザー権限に基づくアクセス制御）
- **Phase 4（合成）**: Gemini 3 Proで最終回答生成
- **外向き API 提供**: HTTP/JSON + SSE でフロントエンドと通信
- **バッチ処理管理**: OCR/Embedding 等の非同期処理を Kafka 経由で管理

#### Librarian（Python）の責務
- **Phase 3（小戦略）**: LangGraphによるLibrarian推論ループ（最大5回推奨）
- **検索戦略立案**: どのような検索を行うべきかの判断
- **終了判定**: 十分な情報が集まったかの評価と停止判断
- **ステートレス**: 会話履歴・キャッシュなし
- **制約**: DB/GCS/Kafka への直接アクセス禁止（Professor 経由のみ）
- **通信**: **gRPC（双方向ストリーミング）** でProfessorと通信、契約: `proto/librarian/v1/librarian.proto`

#### Frontend（Next.js + FSD）の責務
- **Professor の外部 API のみを呼ぶ**: OpenAPI 契約に基づく通信
- **Librarian への直接通信は禁止**: すべて Professor 経由
- **選定エビデンス表示**: Librarian推論ループが選定した根拠箇所を UI で適切に表示
- **会話履歴管理**: Librarianがステートレスのため、フロントエンドがクライアント側で会話履歴を保持

参照: 
- [`../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md`](../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md)
- [`../../eduanimaR_Librarian/docs/README.md`](../../eduanimaR_Librarian/docs/README.md)

---

## バックエンドアーキテクチャとフロントエンドへの影響

### Librarianのステートレス性とフロントエンドへの影響

#### Librarianの特性
- **ステートレス推論サービス**: 会話履歴・キャッシュ等の永続化なし
- **1リクエストで推論完結**: Librarian推論ループは1リクエスト内で完結（最大5回推奨）
- **中断・再開不可**: フロントエンドからの中断・再開は不可
- **通信**: **gRPC（双方向ストリーミング）** でProfessorと通信

#### フロントエンドへの影響

##### 1. 会話履歴の管理
- **クライアント側で保持**: 会話履歴を`localStorage`またはTanStack Query永続化で管理
- **APIリクエストに含める**: Professor APIリクエストに会話履歴を含める必要がある場合、フロントエンドが管理
- **データ構造例**:
```typescript
interface ConversationHistory {
  messages: Array<{
    role: 'user' | 'assistant';
    content: string;
    timestamp: string;
    request_id?: string;
  }>;
  subject_id: string;
}
```

##### 2. Librarian推論ループの扱い
- **ノンストップ実行**: Librarian推論ループは開始後、完了まで中断できない
- **進行状況のみ表示**: `search_loop_progress`イベントでUI更新
- **タイムアウト処理**: 推論時間上限超過時は`LIBRARIAN_TIMEOUT`エラーで通知
- **選定エビデンス表示**: Librarian推論ループが選定した根拠箇所をUI表示

##### 3. キャッシュ戦略
- **結果キャッシュ**: TanStack Queryで推論結果（選定エビデンス含む）をキャッシュ
- **同一質問の再検索**: キャッシュから即座に表示（ユーザー体験向上）

---

## Professor SSEとの統合

### SSEでのリアルタイム配信
Professor SSEでは、以下のイベントをリアルタイム配信します：

| イベントタイプ | 内容 | UI反映 |
|:---|:---|:---|
| `answer_chunk` | 回答断片 | リアルタイムにテキスト追加表示 |
| `evidence_selected` | 選定エビデンス（Librarian推論ループの結果） | 選定エビデンスカードを表示 |
| `search_loop_progress` | Librarian推論ループの中間状態 | プログレスバー更新 |
| `error` | エラー通知 | エラートースト表示 |
| `done` | 完了通知 | SSE接続を閉じる |

### 設計パターン: Librarian推論ループの中間状態をUIに反映

#### `search_loop_progress`イベントの処理
```typescript
eventSource.addEventListener('search_loop_progress', (event) => {
  const data = JSON.parse(event.data);
  
  // プログレスバーを更新
  updateProgressBar({
    current: data.current_retry,
    max: data.max_retries, // 最大5回推奨
    status: data.status, // SEARCHING / COMPLETED / ERROR
  });
  
  // ステータスメッセージを表示
  const statusMessage = {
    SEARCHING: `Librarian推論ループ実行中... (${data.current_retry}/${data.max_retries})`,
    COMPLETED: '推論完了',
    ERROR: 'エラーが発生しました',
  }[data.status];
  
  updateStatusMessage(statusMessage);
});
```

#### UIコンポーネント例
```typescript
// widgets/search-loop-status
export function SearchLoopStatus({ current, max, status }: SearchLoopStatusProps) {
  const progress = (current / max) * 100;
  
  return (
    <Box>
      <LinearProgress variant="determinate" value={progress} />
      <Typography variant="caption">
        {status === 'SEARCHING' && `Librarian推論ループ実行中... (${current}/${max})`}
        {status === 'COMPLETED' && '推論完了'}
        {status === 'ERROR' && 'エラーが発生しました'}
      </Typography>
    </Box>
  );
}
```

---

## TanStack Queryでの状態管理

### Librarian推論結果（選定エビデンス）のキャッシュ

#### キャッシュキー設計
```typescript
// Librarian推論ループ結果（選定エビデンス）をキャッシュ
const queryKey = ['evidence', subjectId, query];

export function useEvidence(subjectId: string, query: string) {
  return useQuery({
    queryKey: ['evidence', subjectId, query],
    queryFn: async () => {
      // Professor API経由でLibrarian推論ループ結果を取得
      const response = await api.searchWithEvidence({ subjectId, query });
      return response.data.evidence; // 選定エビデンス
    },
    staleTime: 5 * 60 * 1000, // 5分
    gcTime: 10 * 60 * 1000, // 10分
  });
}
```

#### 同一質問の再検索時の処理
```typescript
// キャッシュがある場合、即座に表示
export function SearchResults({ subjectId, query }: SearchResultsProps) {
  const { data: evidence, isLoading, isError } = useEvidence(subjectId, query);
  
  if (isLoading) {
    return <SearchLoopStatus status="SEARCHING" />;
  }
  
  if (isError) {
    return <ErrorMessage />;
  }
  
  // キャッシュから選定エビデンスを即座に表示
  return <EvidenceList evidence={evidence} />;
}
```

#### キャッシュの無効化
```typescript
// 新しい質問の場合、キャッシュを無効化
const queryClient = useQueryClient();

function handleNewQuestion(newQuery: string) {
  // 前回の質問のキャッシュを無効化
  queryClient.invalidateQueries({ queryKey: ['evidence', subjectId] });
  
  // 新しい質問を送信（Librarian推論ループ開始）
  searchWithEvidence(newQuery);
}
```

### SSEとTanStack Queryの統合

#### SSEイベントをTanStack Query状態に反映
```typescript
export function useSearchStream(subjectId: string, query: string) {
  const queryClient = useQueryClient();
  
  return useQuery({
    queryKey: ['search', 'stream', subjectId, query],
    queryFn: async () => {
      const eventSource = new EventSource(`/v1/search/stream?query=${query}&subject_id=${subjectId}`);
      
      // Librarian推論ループが選定したエビデンスをキャッシュに反映
      eventSource.addEventListener('evidence_selected', (event) => {
        const data = JSON.parse(event.data);
        
        // TanStack Queryキャッシュに選定エビデンスを反映
        queryClient.setQueryData(['evidence', subjectId, query], (old: Evidence[]) => [
          ...(old || []),
          data.evidence, // 選定エビデンス
        ]);
      });
      
      return new Promise((resolve) => {
        eventSource.addEventListener('done', () => {
          eventSource.close();
          resolve(true);
        });
      });
    },
  });
}
```

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

**補足説明**:
- Frontend は Next.js BFF を経由して Professor（Go Gateway）と通信
- Professor は内部で Librarian（Python）と gRPC で協調
- すべてのデータアクセス（DB/GCS/検索）は Professor が管理

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
