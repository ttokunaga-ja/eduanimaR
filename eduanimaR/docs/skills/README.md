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
- **Phase 2（大戦略）**: タスク分割・停止条件決定
- **Phase 3（物理実行）**: ハイブリッド検索(RRF統合)、動的k値設定、権限強制
- **Phase 4（合成）**: Gemini 3 Proで最終回答生成
- **外向きAPI提供**: HTTP/JSON + SSEでフロントエンドと通信

### Librarian（Python）の責務
- **Phase 3（小戦略）**: LangGraphによるLibrarian推論ループ（最大5回推奨）
- **ステートレス**: 会話履歴・キャッシュなし
- **DB直接アクセス禁止**: Professor経由でのみ検索実行
- **通信**: **gRPC（双方向ストリーミング）** でProfessorと通信

### Frontend責務
- **ProfessorのHTTP/JSON+SSEのみ**: Librarian直接通信禁止
- **選定エビデンス表示**: Librarian推論ループが選定した根拠箇所をUI表示
- **会話履歴管理**: Librarianがステートレスのため、クライアント側で保持

### 検索戦略の詳細（Professor Phase 3物理実行）
- **全文検索（基盤）**: 固有名詞・専門用語に強い
- **pgvector併用**: 同義語・言い換え対応
- **ハイブリッド検索(RRF統合)**: k=60
- **動的k値設定**: N < 1,000: k=5 / N ≥ 100,000: k=20

### Chrome拡張/Web役割分離（Phase 2以降）
- **Phase 2**: 拡張機能のみユーザー登録可能
- **Web版**: 既存ユーザーログイン専用（新規登録不可）
- **未登録ユーザー誘導**: Chrome Web Store/GitHub/導入ガイドへ

参照:
- [`../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md`](../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md)
- [`../../eduanimaR_Librarian/docs/README.md`](../../eduanimaR_Librarian/docs/README.md)

---

## 用語統一（全ドキュメント共通）

| 統一用語 | 説明 | 誤った表現 |
|---------|------|-----------|
| **Librarian推論ループ** | Librarianが実行する推論ロジック | "推論ループ", "検索ループ" |
| **選定エビデンス** | Librarian推論ループが選定した根拠箇所 | "エビデンス", "証拠" |
| **ハイブリッド検索(RRF統合)** | 全文検索+pgvectorのRRF統合 | "ハイブリッド検索", "統合検索" |
| **HTTP/JSON** | Professor ↔ Librarian通信プロトコル | "gRPC" |
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
- **用語統一**: "Librarian推論ループ", "選定エビデンス", "ハイブリッド検索(RRF統合)", "gRPC双方向ストリーミング"（Professor ↔ Librarian）
- **上流ドキュメント参照**: Handbook/Professor/Librarianのドキュメントと整合性を保つ
