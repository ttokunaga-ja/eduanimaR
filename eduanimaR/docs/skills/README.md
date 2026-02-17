# Skills（Frontend / FSD Template）

この `skills/` は、長いドキュメント（docs本体）の要点を **短い実務ルール（Must/禁止/チェックリスト）** に圧縮したものです。

目的：
- AI/人間が「毎回同じ判断」をできるようにする
- 変更頻度が高い/破壊的変更が入りやすい領域の事故を減らす

注意：本リポジトリは 2026 年時点の運用を意図しますが、モデルの知識は固定です。
そのため Skill は「最新仕様の丸暗記」ではなく、**変化に強い判断軸（境界/禁止/確認手順）** を中心に書きます。

## サービスミッション（North Star）

**Mission**: 学習者が、配布資料や講義情報の中から「今見るべき場所」と「次に取るべき行動」を素早く特定できるようにし、理解と継続を支援する

**North Star Metric**: 資料から根拠箇所に到達するまでの時間短縮

**参照**: [`../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md`](../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md)

---

## 上流ドキュメントへの参照

本フロントエンドドキュメントは、以下の上流ドキュメントと整合性を保ちます：

- **Handbook（サービスコンセプト全体）**: [`../../eduanimaRHandbook/README.md`](../../eduanimaRHandbook/README.md)
  - **Mission/Values**: [`../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md`](../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md)
- **Professor（バックエンド Go サービス）**: [`../../eduanimaR_Professor/docs/README.md`](../../eduanimaR_Professor/docs/README.md)
  - **MICROSERVICES_MAP**: [`../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md`](../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md)
  - **ERROR_CODES**: [`../../eduanimaR_Professor/docs/03_integration/ERROR_CODES.md`](../../eduanimaR_Professor/docs/03_integration/ERROR_CODES.md)
- **Librarian（バックエンド Python サービス）**: [`../../eduanimaR_Librarian/docs/README.md`](../../eduanimaR_Librarian/docs/README.md)

フロントエンド開発時は、これらの上流ドキュメントを参照して、サービス全体の責務分担とコンセプトを理解してください。

---

## Professor/Librarian責務境界の理解（最重要）

### Professor（Go）の責務
- **データ守護者（唯一の権限者）**: DB/GCS/Kafka直接アクセス権限を持つ
- **Phase 2: 検索戦略決定**: 
  - 「資料検索を実行すべきか」「ヒアリングすべきか」の判断
  - ヒアリング判断時: Phase 4-Aへ直接遷移（Phase 3スキップ）
  - 検索判断時: 検索戦略・終了条件決定、Librarianへ指示
- **Phase 3 連携**: Librarianへ戦略指示（**gRPC**）、検索実行（Librarian生成クエリ使用）
- **Phase 4: 回答生成**: 
  - **4-A) 意図推測モード**: 曖昧質問への候補選択肢3つ生成
  - **4-B) 最終回答モード**: 検索結果を元に回答生成、SSE配信
- **外向きAPI**: フロントエンドへHTTP/JSON + SSE提供

### Librarian（Python）の責務
- **Phase 3: クエリ生成のみ**: Professorが決定した戦略・終了条件に基づくクエリ生成
- **ステートレス**: 会話履歴・キャッシュなし、1リクエスト内で完結（最大5回試行）
- **禁止事項**: 
  - 検索戦略決定（Professorの責務）
  - 「検索 vs ヒアリング」判断（Professorの責務）
  - DB/GCS直接アクセス
  - フロントエンドとの直接通信

### 通信プロトコル
- **Frontend ↔ Professor**: HTTP/JSON + SSE
  - エンドポイント例: `POST /v1/question`, `POST /v1/question/refine`
  - SSEイベント: `progress`, `clarification`, `evidence`, `answer`, `error`
- **Professor ↔ Librarian**: **gRPC** 双方向ストリーミング
  - proto定義: `eduanimaR_Professor/proto/librarian/v1/librarian.proto`
  - RPC: `Reason(stream ReasoningInput) returns (stream ReasoningOutput)`

### Phase別の責務詳細

#### 推論ループのPhase（Professor ↔ Librarian ↔ Frontend）

| Phase | Professor責務 | Librarian責務 | Frontend責務 |
|-------|--------------|--------------|-------------|
| **Phase 2** | 検索 vs ヒアリング判断、検索戦略決定 | - | プログレス表示「質問を理解中」（プログレスバー） |
| **Phase 4-A** | 意図推測、候補3つ生成（Gemini） | - | 意図選択UI表示、ユーザー選択受付 |
| **Phase 2再実行** | 選択された意図をコンテキストに検索戦略再決定 | - | プログレス表示「質問を理解中」（プログレスバー） |
| **Phase 3** | Librarian gRPC通信、検索実行 | クエリ生成（最大5回試行） | プログレス表示「資料を検索中」（プログレスバー） |
| **Phase 4-B** | 最終回答生成（Gemini）、SSE配信 | - | プログレス表示「回答を生成中」、回答表示、Good/Badフィードバックボタン表示 |

#### リリースPhase（Phase 1-5）

| Phase | 目的 | 実装範囲 | リリース先 | 重要な制約 |
|-------|------|---------|----------|-----------|
| **Phase 1** | バックエンド完成 + Web版完全動作 | Professor完成、Librarian推論ループ統合、Web版固有機能（資料一覧・会話履歴・科目選択UI）、拡張機能実装 | ローカル環境のみ | dev-user固定認証、Web版アップロードUI禁止 |
| **Phase 2** | 拡張機能版作成 + 本番環境デプロイ | SSO認証（Google/Meta/Microsoft/LINE）、拡張機能ZIP配布、Web版未登録ユーザー誘導UI | 本番環境 + ZIP配布 | Web版新規登録禁止、拡張機能のみ登録可能 |
| **Phase 3** | Chrome Web Store公開 | ストア審査対応、プライバシーポリシー | Chrome Web Store | Phase 2から変更なし |
| **Phase 4** | 閲覧中画面の解説機能追加 | HTML・画像取得、Gemini Vision API統合 | 拡張機能・バックエンド | 取得データは短期保存のみ |
| **Phase 5** | 学習計画立案機能 | 小テスト結果分析、学習計画生成 | 未定（構想段階） | プライバシー配慮、匿名化 |

**重要な設計原則**:
- **Phase 2の核心**: 「資料検索を実行すべきか」vs「質問内容をヒアリングすべきか」の判断
- **意図選択後**: Phase 2を再実行（元の質問 + 選択された意図をコンテキストに戦略決定）
- **会話履歴**: previousRequestID で紐付け保持、North Star Metric（到達時間）計測に使用
- **Web版固有機能**: 資料一覧閲覧・会話履歴確認・科目選択UI（すべてPhase 1から提供）
- **拡張機能固有機能**: 自動アップロード・SSO登録・コース判別・画面解説（Phase別に段階実装）

**参照**:
- **Professor詳細**: [`../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md`](../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md)
- **Librarian詳細**: [`../../eduanimaR_Librarian/docs/README.md`](../../eduanimaR_Librarian/docs/README.md)
- **gRPC契約**: [`../../eduanimaR_Professor/proto/librarian/v1/librarian.proto`](../../eduanimaR_Professor/proto/librarian/v1/librarian.proto)
- **Phase 1-5詳細**: [`../../eduanimaRHandbook/04_product/ROADMAP.md`](../../eduanimaRHandbook/04_product/ROADMAP.md)

---

## 用語統一（全ドキュメント共通）

| 統一用語 | 説明 | 誤った表現 |
|---------|------|-----------|
| **Librarian推論ループ** | Librarianが実行する推論ロジック | "推論ループ", "検索ループ" |
| **選定エビデンス** | Librarian推論ループが選定した根拠箇所 | "エビデンス", "証拠" |
| **ハイブリッド検索(RRF統合)** | 全文検索+pgvectorのRRF統合 | "ハイブリッド検索", "統合検索" |
| **gRPC** | Professor ↔ Librarian通信プロトコル | "HTTP/JSON" |
| **データ守護者** | Professor（DB/GCS/Kafka直接アクセス唯一の権限者） | "データ所有者" |
| **動的k値設定** | 件数に応じたk値調整 | "動的k値" |

---

## 最新版の確認（2026-02-11 時点）

このテンプレでは特定プロジェクトの依存を同梱していないため、最新版は外部ソース（npm / nodejs.org）で都度確認します。

取得元：
- npm：`npm view <package> version`
- Node：`curl -fsSL https://nodejs.org/dist/index.json`

最新版（dist-tag: latest、2026-02-11に取得）：

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

Node（公式 index.json、2026-02-11に取得）：
- latest LTS：`v24.13.1`（Krypton）
- latest Current：`v25.6.1`

---

## 読む順（最短）
1. `SKILL_NEXTJS_APP_ROUTER.md`
2. `SKILL_NEXTJS_TURBOPACK.md`
3. `SKILL_MUI_PIGMENT_CSS.md`
4. `SKILL_TANSTACK_QUERY.md`
5. `SKILL_ORVAL_OPENAPI.md`
6. `SKILL_ESLINT_BOUNDARIES.md`
7. `SKILL_TESTING_VITEST.md`
8. `SKILL_TESTING_PLAYWRIGHT.md`
9. `SKILL_TYPESCRIPT.md`
10. `SKILL_NODE_DOCKER_RUNTIME.md`
11. `SKILL_ZOD_RHF_FORMS.md`

---

## 共通の運用原則（Must）
- 迷ったら、実装より先に docs（契約）を更新する
- deep import をしない（Public API）
- “例外追加” で逃げず、構造（境界/責務）を直す
- **Professor/Librarian責務境界を厳守**: Librarian直接通信禁止
- **用語統一**: "Librarian推論ループ", "選定エビデンス", "ハイブリッド検索(RRF統合)", "gRPC"（Professor ↔ Librarian）
- **上流ドキュメント参照**: Handbook/Professor/Librarianのドキュメントと整合性を保つ
