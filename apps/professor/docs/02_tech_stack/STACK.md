# STACK

Last-updated: 2026-02-18

## 技術スタック（2026年2月最新版）

| 項目 | バージョン | リリース日 | 備考・新機能要約 |
| :--- | :--- | :--- | :--- |
| **Go** | **1.25.7** | 2026/02/04 | 最新安定版。セキュリティ修正およびコンパイラ最適化を含む。 |
| **gRPC (google.golang.org/grpc)** | **latest** | - | **Professor ↔ Librarian** の内部通信。Protocol Buffers(.proto)で型安全な契約。双方向ストリーミングに対応。契約: `proto/librarian/v1/librarian.proto` |
| **PostgreSQL** | **18.1** | 2025/10頃~ | `uuidv7()`、非同期I/O (AIO)、B-tree Skip Scanの正式サポート。 |
| **pgvector** | **0.8.1** | 2025/09/04 | HNSWインデックスの構築・検索パフォーマンス向上。反復インデックススキャン対応。 |
| **Atlas** | **v1.0.0** | 2025/12/24 | メジャーリリース到達。Monitoring as Code、Schema Statistics機能追加。 |
| **sqlc** | **1.30.0** | 2025/09/01 | pgx/v5、ENUM配列の対応強化。MySQL/SQLiteエンジンの改善。 |
| **pgx** | **v5.8.0** | 2025/12/26 | Go 1.24+必須化。パイプライン処理の改善、`pgtype.Numeric`の最適化。 |
| **Echo** | **v5.0.1** | 2026/01/28 | Professor の外向きAPI（HTTP/JSON）と **SSE** に使用。 |
| **Kafka (segmentio/kafka-go)** | **latest** | - | IngestJob の publish/consume。非同期ワーカーで OCR/構造化/Embedding準備を実行。 |
| **Google Cloud Run** | - | - | Professor / Librarian の実行基盤（ステートレス）。 |
| **Cloud SQL for PostgreSQL** | - | - | Professor の永続化ストア（pgvector）。 |
| **Google Cloud Storage (GCS)** | - | - | 講義資料の原本ストレージ（Professor のみが直接アクセス）。 |
| **Google Generative AI SDK for Go** | **latest** | - | Professor が Gemini を呼び出す（OCR/構造化/最終生成）。 |
| **slog** | - | - | Goの標準ライブラリ |
| **Testcontainers** | **v0.40.1** | 2025/11/06 | PostgresSQLにてSSL設定（WithSSLSettings）の簡略化、証明書の自動マウントと設定対応 |

## モデル利用（SSOT）

単一モデルで完結させず、用途ごとに最適モデルを使い分ける。

| フェーズ | タスク | モデル | thinking_level |
| :--- | :--- | :--- | :--- |
| Phase 1（Ingestion） | PDF/画像→Markdown化・意味単位チャンク分割（バッチ非同期） | `gemini-3.1-flash-lite-preview` | `minimal` |
| Phase 2（Plan） | 質問意図分析・調査項目（target_items）・停止条件生成 | `gemini-3-flash` | `medium` |
| Phase 3-A（Search: クエリ生成）中間・最終ループ共通 | 検索クエリ生成・リファインメント | `gemini-3-flash` ※1 | `low` ※2 |
| Phase 3-B（Search: 充足性検証）中間ループ | 情報充足性・混同検出・視覚情報確認 | `gemini-3-flash` ※1 | `low` |
| Phase 3-B（Search: 充足性検証）最終ループのみ | 終了判定（偽陽性防止） | `gemini-3-flash` ※1 | `medium` |
| Phase 4（Answer） | Librarianが選定したページのBlobを直接入力し最終回答生成 | `gemini-3-flash` → 昇格候補: `gemini-3.1-pro-preview` ※3 | `low` |

※1 コスト最適化構成（将来）: 3-AのみFlash Liteに変更可。その場合 `LIBRARIAN_THINKING_PLAN=medium` とすること（Flash Lite medium ≒ Flash low の知見に基づく）  
※2 3-Aは最終ループも `low` を維持する。理由: クエリ生成では多様性が重要であり、thinking_levelを上げると「慎重な1択」に収束しやすく逆効果になりうる  
※3 Phase 4 Pro昇格条件は下記「Phase 4 運用ポリシー」を参照

### thinking_level 運用ポリシー

#### ⚠️ モデル別制約・デフォルト値（必ず確認すること）

| モデル | デフォルト | minimal | 備考 |
| :--- | :--- | :--- | :--- |
| `gemini-3.1-flash-lite-preview` | `minimal` | ✅ 可 | |
| `gemini-3-flash` | **`high`** ⚠️ | ✅ 可 | **未設定で高コスト・高レイテンシになる** |
| `gemini-3.1-pro-preview` | **`high`** ⚠️ | ❌ 不可 | **`low` が最低値** |

- **`thinking_level` は全フェーズで必ず明示設定すること**
- `thinking_budget`（旧パラメータ）は Gemini 3系で廃止。`thinking_level` のみを使用し、両者を併用しないこと
- Thinkingトークンは**出力トークンとして課金**される

#### SSEストリーミングへの影響

| thinking_level | TTFT | 実装要件 |
| :--- | :--- | :--- |
| `minimal` | ほぼ遅延なし | 不要 |
| `low` | 軽微な遅延 | 「AIが思考中...」UIプレースホルダーを表示すること |
| `medium` | 標準の1.5〜2倍 | 同上 |
| `high` | 大幅増加 | 教育UIでは原則使用しない |

#### Phase 3 thinking_level 判定ロジック（ループ回数ベース）

```
判定方式: ループ回数ベース（loop_count / max_loops）

3-A クエリ生成:
  全ループ共通 → LIBRARIAN_THINKING_PLAN（デフォルト: low）
  ※ 最終ループも low を維持（多様性重視）

3-B 充足性検証:
  loop_count < max_loops - 1  → LIBRARIAN_THINKING_EVALUATE（デフォルト: low）
  loop_count == max_loops - 1 → LIBRARIAN_THINKING_EVALUATE_FINAL（デフォルト: medium）

採用理由:
  - 実装がシンプル（既存のloop_countを参照するだけ）
  - max_loopsを環境変数で変更しても自動追従する
  - Phase 2の分析結果との責務混在を避けられる
  - 3-Bの偽陽性（不十分→充足と誤判定）は取り返しがつかないため最終判断のみ精度優先
```

### Phase 1 Ingestion 運用ポリシー

- **バッチAPIは使用しない**（リアルタイムAPI統一）
  - 理由: アップロード直後に質問できる即時インデックス化が必要なため
  - 小テストでわからない問題を調べた際、最新資料がヒットしないと満足度に影響する
- 非同期処理はKafkaキューイングで維持（バッチAPIとは別概念）
- `thinking_level=minimal` でデリミタ（`---CHUNK---`）遵守の安定性が高い
  - thinking_levelを上げると自由記述が増えてフォーマット崩れが起きる報告あり
- **ベクトル検索で補えないリスク**への対処
  - リスク: 複雑な数式・表をモデルがスキップする「抽出の放棄（Omission）」
  - 対処: プロンプトで `[図表: 〇〇に関するグラフ]` プレースホルダー指示を含める

### Phase 4 運用ポリシー（Blob直接入力・段階的Pro昇格）

#### Blob入力の設計制約
- **選定ページのみを動的抽出して渡すこと**（Librarianが特定したページ番号群のみ）
- 全ページ渡しは禁止（コスト増 + Lost in the Middle 問題の誘発）
- 200kトークン以下に収めることでProの高額帯（>200k: $4/$18）を回避できる

#### Flash → Pro 昇格条件
以下のいずれかが報告された場合に `PROFESSOR_MODEL_ANSWER=gemini-3.1-pro-preview` へ変更:
- 数式の誤り・ハルシネーションが複数回報告された
- 複数ページにまたがる統合回答の品質が不十分
- 図表の構造解釈ミスが回答に影響した
- 修士〜博士レベルの理論的質問が増加した

#### 部分的Pro昇格（将来の拡張）
難問セッション（全体の10%程度）のみProに昇格させる動的切り替えも選択肢。
実装時は `PROFESSOR_MODEL_ANSWER_HARD=gemini-3.1-pro-preview` 環境変数の追加で対応。

### コスト試算（月額概算・参考値）

| 構成 | 月額概算（USD） | 備考 |
| :--- | :--- | :--- |
| 推奨構成（Ingestion=Flash Lite + QA=Flash） | 約 $80〜90/月 | 1,000セッション・100ファイル想定 |
| 難問10%をPro昇格した場合 | 約 $100〜110/月 | 上記に加算 |
| Phase 3もFlash Liteに最適化した場合 | 約 $50〜60/月 | 3-AにFlash Lite medium使用 |

※ `thinking_level=medium` 設定箇所では出力トークンが約1.5〜2倍になる可能性あり  
※ `gemini-3-flash` のデフォルトは `high` のため、thinking_level未設定時は試算より大幅にコストが増加する

### モデル設定（環境変数）フェーズ別構成

```
Professor:
  PROFESSOR_MODEL_INGESTION        — Phase 1 専用（Flash Lite 固定）
  PROFESSOR_MODEL_PLAN             — Phase 2 専用
  PROFESSOR_MODEL_ANSWER           — Phase 4 専用（Pro昇格スイッチ）
  PROFESSOR_THINKING_INGESTION     — Phase 1: minimal
  PROFESSOR_THINKING_PLAN          — Phase 2: medium
  PROFESSOR_THINKING_ANSWER        — Phase 4: low

Librarian:
  LIBRARIAN_MODEL_SEARCH           — Phase 3 専用
  LIBRARIAN_THINKING_PLAN          — Phase 3-A 全ループ: low
  LIBRARIAN_THINKING_EVALUATE      — Phase 3-B 中間ループ: low
  LIBRARIAN_THINKING_EVALUATE_FINAL — Phase 3-B 最終ループ: medium
```

## 通信スタック（SSOT）
- Frontend ↔ Professor: **HTTP/JSON（OpenAPI）** + **SSE**
- Professor ↔ Librarian: **gRPC（Proto、双方向ストリーミング）**、契約: `proto/librarian/v1/librarian.proto`

## 設計ポリシー
- **UUID + NanoID**: 内部主キーはUUID（推奨: UUIDv7）、外部公開キーはNanoIDを採用し、セキュリティとユーザビリティを両立する。
- **ENUM型の積極採用**: 固定値の管理はPostgreSQL ENUM型を使用し、型安全性・パフォーマンス・可読性を向上させる。
	- **VARCHAR型での固定値管理は禁止**（必ず PostgreSQL ENUM を使う）

## SSOT（Single Source of Truth）
- DBスキーマ: Atlas の `schema.hcl`
- API契約（外向き）: OpenAPI（`docs/openapi.yaml`）
- API契約（内向き）: Proto（`proto/`）
- SQL: `sql/queries/*.sql`（sqlcで生成）

## Post-MVP候補（MVPでは使わない）
- Elasticsearch（検索は Postgres/pgvector を正とする）
- Debezium CDC（Elasticsearch を採用する場合の差分同期手段）

---

## Phase 1で明示的に使わない技術（Phase 2以降に延期）

| 技術 | 延期理由 |
|:---|:---|
| **Elasticsearch** | Phase 1はpgvectorのみで検証（Hybrid検索はPhase 3以降） |
| **Debezium CDC** | Phase 1はリアルタイム同期不要 |
| **SSO認証** | Phase 1は固定dev-userで動作確認（Phase 2でSSO実装） |

Phase 1の技術スタック（確定版）:
- Go 1.25.7
- PostgreSQL 18.1 + pgvector 0.8.1
- Kafka（非同期OCR/Embedding Ingestパイプライン）
- 高速推論モデル（OCR/埋め込み）
- Echo v5.0.1（HTTP API）
- gRPC（Professor ↔ Librarian内部通信、Phase 1から完全実装）

