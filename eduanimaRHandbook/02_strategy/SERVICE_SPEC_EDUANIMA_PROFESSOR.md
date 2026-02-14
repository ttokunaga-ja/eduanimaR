Title: Service Spec — eduanima-professor (Go)
Description: Goマイクロサービス「The Professor」の責務・技術選定・プロセス仕様（実装詳細除く）
Owner: @OWNER
Reviewers: @reviewer1
Status: Draft
Last-updated: 2026-02-14
Tags: strategy, service-spec, gateway, go, security

# eduanima+R: Go Microservice Specification (The Professor)

## 0. サマリ
`eduanima-professor`（Professor）は、システム全体の司令塔/ゲートウェイであり、データの守護者です。
Chrome拡張・Webアプリからの要求を受け、認証/認可を強制しつつ、
- 取り込み（非同期前処理）
- 検索（DBアクセスの独占と物理的制約の強制）
- Librarianとの推論ループのオーケストレーション
- 最終的な学習支援（合成）
を統治します。

## 1. 役割：統治者・教授（The Professor）
Professorのゴール:
- SSO（OAuth/OIDC）に基づき、厳格なユーザー別アクセス制御を強制する
- システム内で唯一、PostgreSQL（pgvector含む）とストレージへ直接アクセスできる境界として機能する
- Librarianが選んだ根拠を用いて、学習支援としての最終出力（要点、参照箇所、学習計画）を合成する

Professorの非ゴール:
- 探索戦略の詳細（検索クエリの試行錯誤）はLibrarianの責務
- クライアント（拡張/WEB）の表示ロジック

## 2. 核心的責務
### 2.1 ゲートウェイ・オーケストレーション（Gateway & Orchestration）
- 外部クライアント（Chrome拡張/Web）からのリクエスト受付
- 認証（OAuth/OIDCトークン検証）と認可（ユーザー/科目/資料のアクセス制御）
- ユーザー要求（質問、取り込み、ロードマップ等）のルーティング

### 2.2 データパイプラインの統治（Data Pipeline Management）
- 資料インジェスト: LMS由来の資料を保存し、解析ジョブを投入する
- 非同期処理: OCR/抽出（Vision Reasoning等）、Markdown化、Embedding生成をワーカーで実行する
- 永続化: 生成した派生データ（テキスト、メタデータ、埋め込み等）をPostgreSQLへ保存する

### 2.3 物理的な「制約」の強制（Physical Constraint Enforcement）
- Librarianからの検索依頼に対し、実行時に必ず `user_id` / `subject_id` / `active` 等の制約をSQL側で強制
- アプリ層でのミスがあっても越権できない構造（Professorが唯一のDB窓口）

### 2.4 最終回答の合成（Final Answer Synthesis）
- Librarianが返した根拠セット（資料ID/ページ/断片）を取得し、LLMで学習支援として合成する
- 出力には参照箇所（ページ、断片）を付与し、透明性/追跡可能性を満たす

注: Professorの最終出力は「学習支援」（資料のどこを見るべきか、理解の要点、学習ロードマップ等）を目的とし、評価・採点の自動化を目的にしない。

## 3. 技術スタックと選定理由
### 3.1 言語・フレームワーク: Go
- 選定理由: 非同期処理、IO並行性、Cloud Run等での運用性、起動速度、コスト効率
- Webフレームワーク: Gin/Echo等（どれを採用するかは別紙で決定）

### 3.2 メッセージング: Kafka
- 選定理由: 資料解析ジョブ（OCR/抽出/Embedding）を非同期で確実に処理する
- 要件: リトライ、DLQ相当、バックプレッシャー制御

### 3.3 DBドライバ: pgx v5+
- 選定理由: PostgreSQL機能（pgvector含む）と相性がよく、性能とプール管理に優れる

### 3.4 AI SDK: Google Generative AI SDK for Go
- 用途: 取込時のVision Reasoning/抽出、最終合成
- 選定理由: 公式SDKでモデル追従性を確保する

### 3.5 DB: GCP上のPostgreSQL + pgvector
- 方針: AlloyDB AIは使用しない。PostgreSQL拡張のpgvectorを採用する。

## 4. インターフェース（外部/内部）
### 4.1 外部（Chrome拡張/Web → Professor）
- 質問/検索: 学習支援の生成要求
- 取込: 資料アップロード、またはLMS検知イベントの受領
- 設定: 科目ID紐付け、アクセス範囲設定（詳細は別紙）

### 4.2 内部（Professor ↔ Librarian）
- Professor→Librarian: `POST /agent/librarian/think` に「質問/履歴/制約/状態」を渡す
- Librarian→Professor: `SEARCH` 要求（検索の実行依頼）または `COMPLETE`（根拠セット返却）

注: 実際の通信形態（ProfessorがLibrarianに呼び出す / LibrarianがProfessorツールを叩く）は実装詳細だが、責務としてはProfessorが検索/取得を独占する。

## 5. プロセスフロー（仕様レベル）
### 5.1 資料解析（Ingestion Loop）
1) 受領: 資料（または資料URL/イベント）を受け取る
2) 原本保存: オブジェクトストレージへ保存
3) ジョブ投入: Kafkaへ解析ジョブを投入
4) ワーカー処理: OCR/抽出→Markdown化→Embedding生成
5) 永続化: PostgreSQLへ保存（科目ID/ユーザーIDなどの権限スコープを必ず付与）

### 5.2 検索・合成（Reasoning Loop）
1) 質問受領: ユーザー質問を受け取る
2) Librarian呼び出し: 1ステップ思考を要求
3) SEARCHの場合: ProfessorがDB検索を実行（subject_id/user_id制約を強制）→結果をLibrarianへ返す
4) COMPLETEの場合: 根拠セットを受け取り、必要な断片をDBから取得
5) 合成: LLMで学習支援として整形し、参照箇所付きで返す

## 6. セキュリティ/監査（必須）
- 認証: OAuth/OIDCのIDトークン検証（JWKS等）
- 認可: リクエスト単位で `user_id` を確定し、DB検索/取得に必ずスコープ条件を付与
- 監査ログ: 取込、検索、生成、エクスポート相当の操作を `request_id` で追跡可能に
- データ境界: Librarianはデータ保管をせず、Professorがデータアクセスを一元化

## 7. 非機能要件（簡易）
- スケール: Cloud Run等で水平スケール可能なステートレスAPI
- 性能: 科目ID等の事前絞り込みを前提に、全文検索を基盤にしつつpgvectorを併用
- 信頼性: 非同期処理はリトライ/冪等性を前提（詳細は別紙）

## 8. Open Questions（要確認）
- （将来）学内IdP連携の要否（Google Workspace / Azure AD / 学内IdP）と、追加する場合のトークン検証方式
- マルチテナント/共有の単位（初期: 個人利用のみ／将来: 科目の資料セットのみ共有。質問履歴・重要箇所マーク等の個人学習履歴は共有しない）
- Cloud SQL以外（Cloud SQL/VM/マネージド）どのPostgreSQLを採用するか
- Kafka運用方式（マネージド/自前）
