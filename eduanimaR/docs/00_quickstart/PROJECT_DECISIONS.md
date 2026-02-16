---
Title: Project Decisions
Description: eduanimaRプロジェクトの技術決定事項とSSO設定のSSOT
Owner: @ttokunaga-ja
Status: Published
Last-updated: 2026-02-15
Tags: frontend, eduanimaR, project-decisions, authentication, api
---

# Project Decisions（SSOT）

Last-updated: 2026-02-15

このファイルは「プロジェクトごとに選択が必要」な決定事項の SSOT。
AI/人間が推測で埋めないために、まずここを埋めてから実装する。

## 基本
- **プロジェクト名**: eduanimaR
- **リポジトリ**: ttokunaga-ja/eduanimaR
- **対象環境**: local / staging / production
- **サービス概要**: 大学LMS資料の自動収集・検索・学習支援を行うChrome拡張機能 + Webアプリ

## 認証（Must）
- **方式**: Cookie（SSO/OAuth 2.0による）
- **SSO対応プロバイダー（Phase 2）**: Google / Meta / Microsoft / LINE
- **Phase 1**: ローカル開発のみ、認証スキップ（固定dev-user使用）
- **セッション保存場所**: Cookie（httpOnly, Secure, SameSite=Lax）
- **401/403 の UI 振る舞い**: ログイン画面へリダイレクト、元ページURLを保持

## API（Must）
- **OpenAPI の取得元**: eduanimaR_Professor（Go）が提供
- **OpenAPI の配置パス（このrepo内）**: `openapi/openapi.yaml`
- **生成物の配置**: `src/shared/api/generated`（固定）
- **バックエンド構成**:
  - **Professor（Go）**: 外向きAPI（HTTP/JSON + SSE）、DB/GCS/Kafka管理、最終回答生成
  - **Librarian（Python）**: LangGraph Agent による検索戦略立案（ProfessorからgRPC経由で呼び出し）

## Next.js（Must）
- **SSR/Hydration**: 原則 Must（学習支援UIの即応性を重視）
- **Route Handler/Server Action の採用方針**: 
  - Server Actions: フォーム送信（設定更新）
  - Route Handler: SSE（リアルタイム回答配信）、Webhook受信
- **キャッシュ戦略（tag/path/revalidate の主軸）**: 
  - 科目・ファイル一覧: `revalidateTag`（資料追加時に無効化）
  - 質問履歴: `no-store`（ユーザー依存データ）
  - 静的UI: `force-cache`（ブランドガイドライン・ヘルプページ）

## FSD（Feature-Sliced Design）
- **採用理由**: マイクロサービス境界（Professor/Librarian）とフロントエンド機能境界を明確に対応付けるため
- **主要Slices**:
  - `entities/subject`: 科目（Professor の subject_id に対応）
  - `entities/file`: 資料ファイル（Professor の GCS URL / metadata に対応）
  - `features/qa-chat`: Q&A（Professor の SSE + Librarian Agent の推論結果）
  - `widgets/file-tree`: 科目別ファイルツリー表示

## i18n（Phase 2以降）
- **対象言語**: 日本語（ja）のみ（初期）
- **翻訳ファイルの置き場**: `src/shared/locales/ja.json`
- **直書き文字列の扱い（lint/CI）**: 警告レベル（段階的に対応）

## 観測性（Must）
- **エラー通知**: Professor と統一のエラーコード体系（`ERROR_CODES.md`）
- **Web Vitals / RUM**: Vercel Analytics（または Google Analytics 4）
- **ログの取り扱い（PII/Secrets）**: 
  - ユーザーID・メールアドレスはハッシュ化
  - 質問内容・資料内容は本番ログに含めない（デバッグ時のみローカル）

## プライバシー・セキュリティ（Must）
- **データ最小化**: Handbook の PRIVACY_POLICY.md に準拠
- **共有範囲**: Phase 1〜4は個人利用のみ（科目内グループ共有は将来検討）
- **質問履歴・学習ログ**: 共有しない（プライバシー保護）
- **CSP**: `SECURITY_CSP.md` に基づく厳格な設定

---

## eduanimaR 固有の前提

### サービスコンセプト
- **Mission**: 学習者が「探す時間を減らし、理解に使う時間を増やせる」学習支援ツール
- **Vision**: 必要な情報が、必要なときに、必要な文脈で見つかり、学習者が自律的に学習を設計できる状態
- **North Star Metric**: 資料から根拠箇所に到達するまでの時間短縮

### 提供形態
- Chrome拡張機能（LMS利用中の介入）
- Webアプリケーション（復習用ダッシュボード）
- **導線統一**: どちらの導線でも同一のログイン体験（SSO/OAuth）と同一の権限境界を維持

### 認証・認可方針
- **Phase 1（ローカル開発）**: 認証スキップ（固定のdev-user使用）
- **Phase 2以降**: SSO認証実装（Google / Meta / Microsoft / LINE）
- **認可**: ユーザー別アクセス制限を厳格に実施（導線（拡張/WEB）に依存しない）

### バックエンド境界
- **Professor（Go）**: データの守護者、APIのSSOT（OpenAPI）、唯一DBに直接アクセス
- **Librarian（Python）**: 推論・検索ループ専門サービス（ステートレス、Professorのみが呼ぶ）
- **フロントエンド**: Professor の OpenAPI（HTTP/JSON + SSE）のみを呼ぶ

### データ境界・プライバシー
- ユーザー別データ分離がデフォルト
- 共有範囲: 将来「科目の資料セット」のみ共有、質問履歴や学習ログは共有しない

### ロードマップ（Phase 1〜4）
- **Phase 1**: ローカル開発、基本的なQ&A機能、資料管理
- **Phase 2**: SSO認証、本番環境デプロイ
- **Phase 3**: 推論ループ（Librarian連携）、高度な検索
- **Phase 4**: 学習計画、進捗管理

---

## eduanimaR 固有の前提（2026-02-15確定）

### サービス境界
- **Professor（Go）**: データ所有者。DB/GCS/Kafka直接アクセス。外向きAPI（HTTP/JSON + SSE）。
- **Librarian（Python）**: 推論特化。Professor経由でのみ検索実行。
- **Frontend（Next.js + FSD）**: Professorの外部APIのみを呼ぶ。Librarianへの直接通信は禁止。

### 認証方式
- **Phase 1**: ローカル開発のみ（dev-user固定、認証UI実装不要）
- **Phase 2**: SSO（Google / Meta / Microsoft / LINE）による本番認証、Web版・拡張機能を同時リリース
- **重要**: Web版からの新規登録は禁止。拡張機能でSSO登録したユーザーのみがログイン可能。

### ファイルアップロード
- **フロントエンドの責務範囲**: フロントエンドはファイルアップロードUIを持たない
- **Phase 1（開発環境）**: 
  - Web版: 外部ツール（curl, Postman等）でProfessor APIへ直接アップロード
  - 拡張機能: 自動アップロード機能の実装と検証（ローカルでのChromeへの読み込み）
- **Phase 2（本番環境）**: Chrome拡張機能による自動アップロードのみ（Phase 1で実装済みの機能を本番適用）
- **禁止事項**: Web版にファイルアップロード機能を実装してはならない

### 自動アップロード機能
- **Phase 1で実装**: Chrome拡張機能のLMS資料自動検知・アップロード機能を完全実装
- **実装内容**:
  - Content Scriptによる資料リンク検知
  - Background Serviceによる定期チェック
  - Professor APIへの自動送信
- **Phase 1での検証方法**: Chromeにローカルで拡張機能を読み込み、Moodleテストサイトで動作確認
- **Phase 2で公開**: Chrome Web Storeへ公開し、本番環境で提供

### データ境界
- user_id / subject_id による厳格な分離（Professor側で強制）
- フロントエンドは物理制約を「信頼」して表示

### 外部API契約（SSOT）
- Professor: `docs/openapi.yaml`（`eduanimaR_Professor/docs/openapi.yaml` が正）
- 生成: Orval（`npm run api:generate`）
- 生成物: `src/shared/api/generated/`（コミット対象）
