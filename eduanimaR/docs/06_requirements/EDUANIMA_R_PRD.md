# eduanima+R 総合要件定義（Web / Chrome Extension）

このドキュメントは、eduanima+R の **フロントエンド（Web）** および **Chrome 拡張** の要件を、バックエンド（Professor/Librarian）の契約と整合する形で統合した SSOT（Single Source of Truth）です。

対象：Phase 1〜5（ロードマップは `RELEASE_ROADMAP.md`）

関連（契約）：
- Professor 外向き OpenAPI（SSE含む）：`../../eduanimaR_Professor/docs/openapi.yaml`
- エラーの標準：`../03_integration/ERROR_HANDLING.md`
- エラーコード：`../03_integration/ERROR_CODES.md`
- Auth/Session（Web）：`../03_integration/AUTH_SESSION.md`

---

## 1. プロジェクト概要

- **プロジェクト名**：eduanima+R
- **コアコンセプト**：「能動的な検索」から「受動的な支援」へのシフト
- **ターゲット**：学部生（当初は個人利用）
- **価値**：資料整理の手間削減、理解度向上、試験対策、弱点克服
- **倫理規定（必須）**：剽窃（カンニング）を助長しない。学習プロセスの効率化・理解の補助を目的とする。

---

## 2. 用語

- **Professor**：外向きAPI（HTTP/JSON + SSE）を提供する Go サービス（司令塔）。
- **Librarian**：検索・評価ループを担う Python サービス（内部は gRPC）。
- **SSE**：Server-Sent Events。回答生成や進捗をストリーミング配信。
- **Subject（科目）**：物理フィルタリングの必須スコープ。Professor API では `subject_id` が必須。

---

## 3. 共通UX要件（全フェーズ）

### 3.1 UI/UX コンセプト

- 学習を妨げないクリーンなインターフェース
- 情報の **透明性**（根拠/ソース提示）と **即時性**（ストリーミング）を最優先

### 3.2 透明性（Source 提示）

- AI回答には必ず「根拠（Source）」を添付する
- Source は **クリック可能** で、ユーザーが即座に確認できる
- Source 表示最小要素（推奨）：
  - 表示名（例：ファイル名）
  - パス/URL
  - 参照箇所（ページ番号/見出し/チャンクIDなど、バックエンドが提供可能な範囲）

### 3.3 即時性（Streaming）

- 解析状況・回答生成は SSE でリアルタイム表示する
- ストリーミングのイベント契約は「UI都合で壊さない」（安定化）

---

## 4. 認証・ユーザー管理（Auth）

### 4.1 SSO（必須）

- 対応プロバイダ（要件）：Google / Meta / Microsoft / LINE
- Web（Next.js）では、セッションは原則 **HttpOnly Cookie**（`AUTH_SESSION.md`）

### 4.2 Chrome 拡張での認証（要件）

- 拡張は Web の Cookie セッションに依存しない（ブラウザ環境差・ドメイン制約のため）
- 拡張は **Bearer JWT**（Professor OpenAPI の `bearerAuth`）でAPIアクセスする
- トークンの保管は最小化（短命 access token、保存先は `chrome.storage.session` を優先）

拡張の詳細は `../03_integration/CHROME_EXTENSION_BACKEND_INTEGRATION.md` を正とする。

---

## 5. 機能要件（フロントエンド）

### 5.1 Q&A（チャットUI）

- チャット形式で質問・回答を表示
- 回答はストリーミング（SSE）で逐次レンダリング
- 回答には Source が付く（3.2）

### 5.2 科目・資料管理

- 科目を作成できる
- 科目を削除できる（誤操作防止の確認を含む）
- 科目ごとに資料をツリー表示できる
- 手動アップロードでは、科目をドロップダウンで指定できる

補足（Web UI 要件）：
- Top Global Bar の **科目選択プルダウン** から、科目の **新規追加/削除** を行えること

### 5.3 履歴（スレッド）

- 質問履歴（スレッド）を保存し、一覧/詳細で閲覧できる

### 5.4 サポート（問い合わせ/不具合）

- 不具合申告・お問い合わせフォームを常設
- 送信時には requestId/traceId 等の相関情報を添付できる（可能なら）

---

## 6. Chrome 拡張固有要件

### 6.1 形態（Phase 2〜）

- Popup + Sidepanel
- LMS 上にフローティングボタン（その場で質問）

### 6.2 自動検知・自動保存（Auto-Ingestion）

- LMS の DOM を監視し、新規配布資料（PDF/PPT等）を検知
- Go サーバーへ自動転送し、解析パイプライン（Kafka等）へ投入
- ユーザーの資料収集コストをゼロに近づける

### 6.3 小テスト HTML 解析（Phase 4）

- 小テスト画面の HTML から、問題文/選択肢/正誤結果を抽出
- 誤答傾向を分析し、「見直すべき資料」を提示

### 6.4 コンテキスト自動認識（Phase 5）

- 現在開いているページ内容を解析し、科目/トピックを自動推定
- 科目指定なしで最適な検索スコープを適用し支援を開始

---

## 7. バックエンド整合（フロントが前提とする契約）

### 7.1 Professor（外向き）

- `POST /v1/questions`：推論セッション開始（`202` を返し、SSE で進捗/回答を配信）
- `GET /v1/questions/{request_id}/events`：SSE ストリーム
- `POST /v1/materials`：資料アップロード（`multipart/form-data`、`202`）

契約の正：`../../eduanimaR_Professor/docs/openapi.yaml`

### 7.2 物理フィルタリング（Subject ID）

- `subject_id` は必須（バックエンドで強制）
- フロントは、ユーザーが「どの科目に対する質問か」を常に明示できる UX を提供する
- Phase 5 では、拡張が `subject_id` を自動推定しバックグラウンド適用する

運用上の前提：
- 科目の作成/削除はフロントから行い、Professor が所有者制約を強制する

---

## 8. 非機能要件（フロント観点）

### 8.1 パフォーマンス

- ストリーミングで「待ち」を見せる（無反応時間を減らす）
- データ取得はキャッシュ（TanStack Query）で重複リクエストを抑制

### 8.2 セキュリティ

- Web：HttpOnly Cookie セッション + CSRF 方針を固定（`AUTH_SESSION.md`）
- 拡張：トークン最小化、権限（Host permissions）最小化
- 収集データ（LMS DOM/小テスト結果）はプライバシー観点で最小化し、ユーザー意図のない収集をしない

### 8.3 観測性/サポート

- 失敗は分類してUI/運用に載せる（`ERROR_HANDLING.md`）
- 問い合わせには相関ID（requestId/traceId）を添付できる設計を推奨
