# Docs Portal（Frontend / FSD Template）

Last-updated: 2026-02-16

この `docs/` 配下は、Next.js（App Router）+ FSD（Feature-Sliced Design）での開発を「契約（運用ルール）」として固定するためのドキュメント集です。

目的：
- 判断のぶれ（人間/AI）を減らす
- 依存境界・契約駆動・運用の事故を先に潰す
- "本番だけ壊れる" を再現可能な手順に落とす

## サービス全体のコンセプト

eduanimaRは、学習者が「探す時間を減らし、理解に使う時間を増やせる」学習支援ツールです。大学LMS資料の自動収集・検索・学習支援を、Chrome拡張機能とWebアプリで提供します。

### Mission & North Star（詳細は Handbook 参照）
- **Mission**: 学習者が、配布資料や講義情報の中から「今見るべき場所」と「次に取るべき行動」を素早く特定できるようにし、理解と継続を支援する
- **Vision**: 必要な情報が、必要なときに、必要な文脈で見つかり、学習者が自律的に学習を設計できる状態を当たり前にする
- **North Star Metric**: 資料から根拠箇所に到達するまでの時間短縮（主要タスク完了時間の削減）
- **補助指標**: 根拠提示率、目標行動明確化率

**参照**: [`../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md`](../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md)、[`../../eduanimaRHandbook/05_goals/OKR_KPI.md`](../../eduanimaRHandbook/05_goals/OKR_KPI.md)

### プロダクト4大原則
eduanimaRは以下の4大原則に基づき設計されています：

1. **学習支援目的（Learning Support First）**: 学習者の発見・理解・計画を支援する（自動回答生成ではない）
2. **データ最小化（Data Minimization）**: 必要最小限のデータのみ収集・保持する
3. **厳格なアクセス制御（Strict Access Control）**: SSO基盤、ユーザー別データ分離、デフォルト非共有
4. **透明性（Traceability & Explainability）**: すべての重要な操作をログ記録し、ユーザーが「なぜ」を理解できるようにする

**参照**: [`../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md`](../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md)

### Professor / Librarian の責務境界（SSOT）

本システムは **2サービス構成** です。DB/GCS/検索インデックスへの直接アクセス権限は Professor のみに付与します（最重要不変条件）。

#### Professor（Go）の責務
**役割**: データの守護者、システムの司令塔、学習支援の最終執筆者

- **認証・認可**: SSO（OAuth/OIDC）トークン検証、ユーザー/科目/資料のアクセス制御
- **Phase 2（大戦略）**: Gemini 3 Flashで「タスク分割（調査項目：WHAT）」「停止条件」「コンテキスト」を生成
- **Phase 3（物理実行）**: 
  - Librarianからの検索依頼（gRPC）を受け、**ハイブリッド検索（RRF統合）** を物理的に実行
  - **動的k値設定**: 母数N（全チャンク数）と`retry_count`に基づき取得件数を調整
  - **除外制御**: `seen_chunk_ids`でDB層で物理的に既読除外
  - **権限強制**: `subject_id`/`user_id`/`is_active` 等の WHERE を SQL 層で必ず強制
- **Phase 4（合成）**: 選定エビデンスをもとにGemini 3 Proで学習支援としての最終回答を生成
- **データ管理**: PostgreSQL（pgvector含む）、GCS、Kafka への**唯一の直接アクセス権限**を持つ
- **バッチ処理**: OCR/Embedding 生成を Gemini Batch API + Kafka で管理

**参照**: [`../../eduanimaR_Professor/docs/README.md`](../../eduanimaR_Professor/docs/README.md)、[`../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md`](../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md)

#### Librarian（Python）の責務
**役割**: 探索・根拠収集の専門家（DB-less、ステートレス）

- **Phase 3（小戦略・ループ制御）**: LangGraphによる自律的検索ループ（最大5回推奨）
  - **Plan/Refine**: 検索クエリ生成（`keyword_list` + `semantic_query`）、反省/再試行
  - **Search Tool**: ProfessorのgRPCサービス経由で検索実行を依頼
  - **Evaluate**: 検索結果から選定エビデンス（`evidence_snippets`）を抽出、`temp_index`を使用
  - **Route**: 停止条件判定（`COMPLETE` / `CONTINUE` / `ERROR`）
- **ステートレス設計**: 会話履歴・キャッシュなし（1リクエスト内で推論完結）
- **DB/GCS 直接アクセス禁止**: すべて Professor 経由でデータ取得
- **LLMには実IDを見せない**: Professorが割り当てた`temp_index`のみ扱う

**参照**: [`../../eduanimaR_Librarian/docs/README.md`](../../eduanimaR_Librarian/docs/README.md)、[`../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_LIBRARIAN.md`](../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_LIBRARIAN.md)

#### Frontend（Next.js + FSD）の責務
**役割**: UI/UX 提供、Professor API との統合

- **Professor API のみを呼ぶ**: HTTP/JSON + SSE でバックエンドと通信（OpenAPI/Orval 生成）
- **Librarian への直接通信は禁止**: すべて Professor 経由
- **SSE リアルタイム表示**: 推論状態（thinking/searching/evidence/answer）をユーザーに可視化
- **Chrome拡張機能**: LMS資料の自動検知・アップロード、ユーザー登録（Phase 2以降）
- **Web版**: 既存ユーザーの閲覧専用（Phase 2以降、新規登録は拡張機能でのみ可能）

#### 通信プロトコル
- **Frontend ↔ Professor**: HTTP/JSON（OpenAPI） + SSE（リアルタイム回答配信）
- **Professor ↔ Librarian**: **gRPC（双方向ストリーミング、契約: `proto/librarian/v1/librarian.proto`）**

#### ハイブリッド検索（RRF統合）の詳細設計

本システムの検索戦略は、**全文検索を基盤** とし、必要に応じて **ベクトル検索（pgvector）** を併用するハイブリッドアプローチです。

**検索戦略の基本方針**（参照: [`../../eduanimaRHandbook/02_strategy/TECHNICAL_STRATEGY.md`](../../eduanimaRHandbook/02_strategy/TECHNICAL_STRATEGY.md)）:
- **ベースは全文検索**: 固有名詞・専門用語・数式に強い（講義資料の特性に適合）
- **pgvector併用**: 同義語・言い換え・抽象度の高い問いに対応
- **事前OCR**: テキスト化で全文検索の費用対効果を最大化

**実行主体**: Professor（Go）の検索ツール
**入力**: LibrarianからのgRPCリクエスト
```json
{
  "keyword_list": ["決定係数", "定義"],
  "semantic_query": "決定係数の定義と計算式",
  "exclude_ids": ["chunk_001", "chunk_005"]
}
```

**処理フロー**:
1. **並列検索**: BM25（全文検索、PostgreSQL `tsvector`） + pgvector（ベクトル検索、コサイン類似度）
2. **RRF統合**: Reciprocal Rank Fusion（k=60）で順位ベースに統合
   - 統合式: `Score = 1/(60 + Rank_vector) + 1/(60 + Rank_keyword)`
   - 目的: BM25スコア（0〜∞）とコサイン類似度（0〜1）の単位差を吸収
3. **動的k値**: 母数N（全チャンク数）と`retry_count`に基づき取得件数を調整
   | 母数N | k（初回） | k（2回目） | k（3回目以降） |
   |:---:|:---:|:---:|:---:|
   | N < 1,000 | 5 | 10 | 15 |
   | 1,000 ≤ N < 100,000 | 10 | 20 | 30 |
   | N ≥ 100,000 | 20 | 40 | 50 |
4. **除外制御**: `WHERE id NOT IN (exclude_ids)`をSQL層で強制（既読チャンクの除外）
5. **権限強制**: `subject_id`/`user_id`/`is_active` を SQL WHERE で必ず強制

**返却**: `temp_index`付きチャンクリスト（最大k件、Markdown断片 + メタデータ）

**参照**: [`00_quickstart/PROJECT_DECISIONS.md`](00_quickstart/PROJECT_DECISIONS.md)（検索パラメータの決定事項）、[`../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md`](../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md)（Phase 3検索ループの設定指針）

### 上流ドキュメントへの参照

#### eduanimaRHandbook（サービスコンセプト全体）
- Handbook全体: [`../../eduanimaRHandbook/README.md`](../../eduanimaRHandbook/README.md)
- **01_philosophy（哲学・価値観）**:
  - ミッション・ビジョン・プロダクト原則: [`../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md`](../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md)
  - プライバシーポリシー: [`../../eduanimaRHandbook/01_philosophy/PRIVACY_POLICY.md`](../../eduanimaRHandbook/01_philosophy/PRIVACY_POLICY.md)
- **02_strategy（戦略）**:
  - リーンキャンバス: [`../../eduanimaRHandbook/02_strategy/LEAN_CANVAS.md`](../../eduanimaRHandbook/02_strategy/LEAN_CANVAS.md)
  - Professor サービス仕様: [`../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_PROFESSOR.md`](../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_PROFESSOR.md)
  - Librarian サービス仕様: [`../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_LIBRARIAN.md`](../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_LIBRARIAN.md)
- **03_customer（顧客理解）**:
  - ペルソナ: [`../../eduanimaRHandbook/03_customer/PERSONAS.md`](../../eduanimaRHandbook/03_customer/PERSONAS.md)
  - カスタマージャーニー: [`../../eduanimaRHandbook/03_customer/CUSTOMER_JOURNEY.md`](../../eduanimaRHandbook/03_customer/CUSTOMER_JOURNEY.md)
- **04_product（プロダクト）**:
  - ブランドガイドライン: [`../../eduanimaRHandbook/04_product/BRAND_GUIDELINES.md`](../../eduanimaRHandbook/04_product/BRAND_GUIDELINES.md)
  - ビジュアルアイデンティティ: [`../../eduanimaRHandbook/04_product/VISUAL_IDENTITY.md`](../../eduanimaRHandbook/04_product/VISUAL_IDENTITY.md)
  - ロードマップ: [`../../eduanimaRHandbook/04_product/ROADMAP.md`](../../eduanimaRHandbook/04_product/ROADMAP.md)
- **05_goals（目標・指標）**:
  - OKR/KPI: [`../../eduanimaRHandbook/05_goals/OKR_KPI.md`](../../eduanimaRHandbook/05_goals/OKR_KPI.md)

#### バックエンドサービス実装
- バックエンド Professor 実装: [`../../eduanimaR_Professor/docs/README.md`](../../eduanimaR_Professor/docs/README.md)
  - マイクロサービス構成: [`../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md`](../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md)
  - エラーコード体系: [`../../eduanimaR_Professor/docs/03_integration/ERROR_CODES.md`](../../eduanimaR_Professor/docs/03_integration/ERROR_CODES.md)
- バックエンド Librarian 実装: [`../../eduanimaR_Librarian/docs/README.md`](../../eduanimaR_Librarian/docs/README.md)
  - Librarian詳細仕様: [`../../eduanimaR_Librarian/docs/01_architecture/EDUANIMA_LIBRARIAN_SERVICE_SPEC.md`](../../eduanimaR_Librarian/docs/01_architecture/EDUANIMA_LIBRARIAN_SERVICE_SPEC.md)

---

## Quickstart（最短で開発開始）
0. `00_quickstart/QUICKSTART.md`（30分で着手できる状態にする）
1. `00_quickstart/PROJECT_DECISIONS.md`（プロジェクト固有の決定事項SSOT）

**重要な前提（Phase構成とロードマップ）**:

本プロジェクトは段階的リリースを前提とし、Phase 1〜4で機能を積み上げます。**Phase 2で Chrome拡張機能とWebアプリを同時リリース** し、Chrome拡張機能が主要チャネル、Webアプリが補助チャネルとなります。

詳細は [`../../eduanimaRHandbook/04_product/ROADMAP.md`](../../eduanimaRHandbook/04_product/ROADMAP.md) を参照してください。

- **Phase 1（開発環境 + Librarian統合）**: 
  - ローカルでの動作確認のみ
  - 認証なし（dev-user固定）
  - **Librarian推論ループの実装と検証（必須要件）**
  - Professor → Librarian（HTTP/JSON）→ Professor のフロー確認
  - 自動アップロード機能の実装と検証
  - Web版: curlやPostmanでAPIテスト + SSE動作確認
  - 拡張機能: Chromeにローカル読み込みで動作確認
  
- **Phase 2（本番環境・同時リリース）**:
  - SSO認証実装（Google/Meta/Microsoft/LINE）
  - Chrome Web Storeへ公開（非公開配布）
  - Webアプリの本番デプロイ
  - Librarian連携の本番適用
  - **Web版からの新規登録は禁止、拡張機能でのみユーザー登録可能**
  - **Web版で新規ユーザーのログイン試行を検知した場合、以下へ誘導**：
    1. Chrome Web Store（拡張機能公式ページ）
    2. GitHubリリースページ（代替ダウンロード）
    3. 公式導入ガイド・解説ブログ
  
- **Phase 3以降（将来）**:
  - 学習ロードマップ生成（Learning Support）
  - 小テストHTML解析（Feedback Loop）
  - コンテキスト自動認識サポート（Seamless Experience）
  
- **ファイルアップロード（重要）**: 
  - **フロントエンドにUIを実装してはならない**
  - Phase 1: API直接呼び出し（curl/Postman） + 拡張機能実装
  - Phase 2: 拡張機能の自動アップロードのみ（Phase 1で実装済みの機能を本番適用）

## まず読む（最短ルート）
1. **プロジェクト固有の前提**: `00_quickstart/PROJECT_DECISIONS.md` ← **最優先**
2. 技術スタック（SSOT）：`02_tech_stack/STACK.md`

## 認証とユーザー登録の境界（Phase 2）

### ユーザー登録フロー（Chrome拡張機能とWebの役割分離）
- **新規登録**: Chrome拡張機能でのSSO認証のみ許可
- **既存ユーザーのログイン**: Web版でも可能（拡張機能で登録済みのユーザーのみ）

### Web版での未登録ユーザー対応
Web版でSSO認証後、未登録ユーザーと判定された場合：
1. **登録不可の通知**を表示
2. **拡張機能ダウンロードページへ誘導**（優先順位順に表示）:
   - Chrome Web Store: `https://chrome.google.com/webstore/detail/[extension-id]`
   - GitHub Releases: `https://github.com/[org]/[repo]/releases`
   - 公式導入ガイド: `[ブログURL]` または `[公式ドキュメント]`
3. **誘導UI**:
   - タイトル: 「eduanimaRをご利用いただくには、Chrome拡張機能のインストールが必要です」
   - 説明: 「Web版は既存ユーザーのログイン専用です。新規登録は拡張機能から行ってください。」
   - ボタン: 「拡張機能をインストール」（Chrome Web Storeへリンク）
   - 補足リンク: 「GitHubからダウンロード」「導入ガイドを見る」

### バックエンド（Professor）との連携
- Professor API: `POST /auth/login` が `user_not_found` を返した場合
- フロントエンド: 拡張機能誘導画面へルーティング
- エラーコード: `AUTH_USER_NOT_REGISTERED`（`ERROR_CODES.md`に追加）

---

## エラーコードと品質原則の整合性

eduanimaRのエラーハンドリングは、Handbookで定義された品質原則（追跡可能性・説明可能性・透明性）に基づきます。

### 品質原則との対応（SSOT: [`../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md`](../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md)）

#### 1. 追跡可能性（Traceability）
- **原則**: 重要な処理・アクセスは後から検証できる形で記録し、ユーザーが何が起きたか理解できるようにする
- **実装**:
  - すべてのAPIリクエストに`request_id`を付与（`X-Request-ID`ヘッダー）
  - Professor → Librarian推論ループでも`request_id`を伝播
  - エラーレスポンスに`request_id`を含める
  - ログ横断検索で原因調査が可能

#### 2. 説明可能性（Explainability）
- **原則**: ユーザーが「なぜそうなったか」を理解できる情報を提供
- **実装**:
  - エラーメッセージは機械可読（`code`）と人間可読（`message`）を分離
  - 選定エビデンスには「なぜ選ばれたか」（`why_relevant`）を付与
  - 検索結果0件時には「検索条件を緩める」などの提案を表示

#### 3. 透明性（Transparency）
- **原則**: 何を保存し、何に使い、どこへ送信されるかを明確にする
- **実装**:
  - SSEイベントで推論進行状態をリアルタイム表示（thinking/searching/evidence/answer）
  - 参照元資料へのクリッカブルリンク（GCS署名付きURL + ページ番号）
  - データ取り扱い方針をプライバシーポリシーで明示

### エラーコード体系（SSOT: `03_integration/ERROR_CODES.md`、Professor: [`../../eduanimaR_Professor/docs/03_integration/ERROR_CODES.md`](../../eduanimaR_Professor/docs/03_integration/ERROR_CODES.md)）

**共通レスポンス形式**:
```json
{
  "error": {
    "code": "USER_NOT_FOUND",
    "message": "User not found",
    "details": { "field": "user_id" },
    "request_id": "abc-123-def"
  }
}
```

**主要エラーコードとUI対応**:
| code | HTTP | 意味 | フロントエンド対応例 |
| --- | --- | --- | --- |
| `VALIDATION_FAILED` | 400 | 入力が不正 | フォームエラーを表示（Zod） |
| `UNAUTHORIZED` | 401 | 認証なし/無効 | ログイン画面へリダイレクト |
| `FORBIDDEN` | 403 | 権限なし | 権限不足のメッセージ表示 |
| `NOT_FOUND` | 404 | リソース無し | 404ページ表示 |
| `CONFLICT` | 409 | 競合（重複/状態不整合） | 再試行を促す |
| `RATE_LIMITED` | 429 | レート制限 | リトライ待機時間を表示 |
| `INTERNAL` | 500 | 想定外エラー | 汎用エラーページ |
| `DEPENDENCY_UNAVAILABLE` | 503 | 依存サービス障害 | メンテナンス中表示 |
| `AUTH_USER_NOT_REGISTERED` | 403 | 認証済み・未登録 | 拡張機能誘導画面表示 |

---

## まず読む（最短ルート）（続き）
3. FSD 全体像：`01_architecture/FSD_OVERVIEW.md`
4. レイヤー境界とバックエンド対応：`01_architecture/FSD_LAYERS.md`
5. Slices とバックエンド境界の対応：`01_architecture/SLICES_MAP.md`
6. 認証・セッション管理：`03_integration/AUTH_SESSION.md` ← **Phase 2以降の必読**
7. データ取得の契約（DAL）：`01_architecture/DATA_ACCESS_LAYER.md`
8. API 契約運用（バックエンドとの通信）：`03_integration/API_CONTRACT_WORKFLOW.md`
9. API 生成（Orval）：`03_integration/API_GEN.md`
10. バージョニング/廃止：`03_integration/API_VERSIONING_DEPRECATION.md`
11. エラー処理の標準：
   - `03_integration/ERROR_HANDLING.md`
   - `03_integration/ERROR_CODES.md`
12. キャッシュ/再検証：`01_architecture/CACHING_STRATEGY.md`
13. セキュリティ（CSP/ヘッダー）：`03_integration/SECURITY_CSP.md`
14. 運用（最小）：
    - `05_operations/OBSERVABILITY.md`
    - `05_operations/RELEASE.md`
    - `05_operations/PERFORMANCE.md`

## 観測性とrequest_id追跡

### request_idの伝播（エビデンスのトレース）
eduanimaRでは、リクエストの追跡可能性を確保するため、以下の経路で`request_id`を伝播します：

1. **フロントエンド → Professor**: Professor APIリクエストに`X-Request-ID`ヘッダーを含める
2. **Professor → Librarian**: Librarian推論ループ呼び出し（HTTP/JSON）時に`request_id`を渡す
3. **Professor → フロントエンド**: SSEイベントおよびレスポンスに`request_id`を含める

### トレース方法（説明責任の担保）
- **ログ検索**: `request_id`でProfessor/Librarianのログを横断検索
- **エラー追跡**: エラー発生時、`request_id`を含むログで原因調査
- **パフォーマンス分析**: `request_id`単位でリクエスト処理時間を計測
- **エビデンス検証**: 「なぜこの資料が選ばれたか」を`request_id`で追跡可能

**品質原則との対応**:
- **追跡可能性**: 問題発生時に原因を特定できる（Handbook 品質原則4）
- **説明可能性**: エビデンス選定理由を後から検証できる

詳細は `05_operations/OBSERVABILITY.md` および [`../../eduanimaR_Professor/docs/05_operations/OBSERVABILITY.md`](../../eduanimaR_Professor/docs/05_operations/OBSERVABILITY.md) を参照。

---

## Architecture
- FSD：
  - `01_architecture/FSD_OVERVIEW.md`
  - `01_architecture/FSD_LAYERS.md`
  - `01_architecture/SLICES_MAP.md`
- Data Access / Cache：
  - `01_architecture/DATA_ACCESS_LAYER.md`
  - `01_architecture/CACHING_STRATEGY.md`
- UI設計：`01_architecture/COMPONENT_ARCHITECTURE.md`
- A11y（最小契約）：`01_architecture/ACCESSIBILITY.md`
- FSD ツール運用：`01_architecture/FSD_TOOLING.md`
- レジリエンス（FE版）：`01_architecture/RESILIENCY.md`

---

## Tech Stack
- `02_tech_stack/STACK.md`
- `02_tech_stack/MUI_PIGMENT.md`
- `02_tech_stack/SSR_HYDRATION.md`
- `02_tech_stack/STATE_QUERY.md`
- `02_tech_stack/SERVER_ACTIONS.md`
- `02_tech_stack/ROUTING_UX_CONVENTIONS.md`

---

## Integration（契約/境界）
- API 生成：`03_integration/API_GEN.md`
- API 契約ワークフロー：`03_integration/API_CONTRACT_WORKFLOW.md`
- バージョニング/廃止：`03_integration/API_VERSIONING_DEPRECATION.md`
- エラー形式/扱い：`03_integration/ERROR_HANDLING.md`
- エラーコード：`03_integration/ERROR_CODES.md`
- CSP/ヘッダー：`03_integration/SECURITY_CSP.md`
- Auth/Session：`03_integration/AUTH_SESSION.md`
- i18n/Locale（必要な場合）：`03_integration/I18N_LOCALE.md`
- Docker 環境：`03_integration/DOCKER_ENV.md`

---

## Testing
- 戦略：`04_testing/TEST_STRATEGY.md`
- ピラミッド：`04_testing/TEST_PYRAMID.md`
- 性能（フロント）：`04_testing/PERFORMANCE_LOAD_TESTING.md`

---

## Operations
- 観測性：`05_operations/OBSERVABILITY.md`
- 性能：`05_operations/PERFORMANCE.md`
- リリース：`05_operations/RELEASE.md`
- CI/CD：`05_operations/CI_CD.md`
- SLO/アラート：`05_operations/SLO_ALERTING.md`
- Secrets/Key：`05_operations/SECRETS_KEY_MANAGEMENT.md`
- Identity/Zero Trust：`05_operations/IDENTITY_ZERO_TRUST.md`
- 脆弱性運用：`05_operations/VULNERABILITY_MANAGEMENT.md`
- サプライチェーン：`05_operations/SUPPLY_CHAIN_SECURITY.md`
- インシデント：`05_operations/INCIDENT_POSTMORTEM.md`

---

## Requirements（要件管理）
- ポータル：`06_requirements/README.md`
- ページ要件：`06_requirements/pages/`
- コンポーネント要件：`06_requirements/components/`

---

## Skills（Agent向け：短い実務ルール）
- `skills/README.md`

運用の基本：
- "迷ったらコードではなくドキュメントを更新して契約を変える"
- "例外は増やさず、境界の切り方を見直す"

---

## バックエンドドキュメントとの関係

���ロントエンドは **Professor（Go）** を通じてバックエンドと通信します。

- バックエンド全体の責務と契約：`../eduanimaR_Professor/docs/README.md`
- バックエンドとフロントエンドの対応関係：`01_architecture/FSD_LAYERS.md` 内の対応表を参照
- API契約の詳細：`03_integration/API_CONTRACT_WORKFLOW.md` および `../eduanimaR_Professor/docs/03_integration/API_GEN.md`
### バックエンド設計への直接リンク（SSOT）
- **Professor 全体の責務と契約**: [`../../eduanimaR_Professor/docs/README.md`](../../eduanimaR_Professor/docs/README.md)
- **サービス境界（MICROSERVICES_MAP）**: [`../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md`](../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md)
- **DB スキーマ設計**: [`../../eduanimaR_Professor/docs/01_architecture/DB_SCHEMA_DESIGN.md`](../../eduanimaR_Professor/docs/01_architecture/DB_SCHEMA_DESIGN.md)
- **DB テーブル定義**: [`../../eduanimaR_Professor/docs/01_architecture/DB_SCHEMA_TABLES.md`](../../eduanimaR_Professor/docs/01_architecture/DB_SCHEMA_TABLES.md)
- **ENUM 参照**: DB設計ドキュメント内に記載（StatusEnum、RoleEnum 等）
- **Professor の Clean Architecture**: [`../../eduanimaR_Professor/docs/01_architecture/CLEAN_ARCHITECTURE.md`](../../eduanimaR_Professor/docs/01_architecture/CLEAN_ARCHITECTURE.md)
- **Librarian 責務詳細**: [`../../eduanimaR_Librarian/docs/README.md`](../../eduanimaR_Librarian/docs/README.md)
