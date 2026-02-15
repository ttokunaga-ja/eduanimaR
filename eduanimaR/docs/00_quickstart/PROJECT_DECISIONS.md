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
- **SSO対応プロバイダー（Phase 2以降）**: Google / Meta / Microsoft / LINE
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
  - Server Actions: フォーム送信（ファイルアップロード、設定更新）
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
  - `features/file-upload`: 資料アップロード（Professor の IngestJob → Kafka経由）
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

## eduanimaR 固有の前提（2026-02-15確定）

### サービス境界
- **Professor（Go）**: データ所有者。DB/GCS/Kafka直接アクセス。外向きAPI（HTTP/JSON + SSE）。
- **Librarian（Python）**: 推論特化。Professor経由でのみ検索実行。
- **Frontend（Next.js + FSD）**: Professorの外部APIのみを呼ぶ。Librarianへの直接通信は禁止。

### 認証方式
- Phase 1: ローカル開発のみ（dev-user固定）
- Phase 2以降: SSO（Google / Meta / Microsoft / LINE）
- **重要**: Web版からの新規登録は禁止。拡張機能でSSO登録したユーザーのみがログイン可能。

### ファイルアップロード
- **本番環境**: Chrome拡張機能による自動アップロードのみ許可
- **開発環境**: 手動アップロード機能は開発確認用途のみで存在
- **禁止事項**: Web版での手動アップロード機能を本番環境で有効化してはならない

### データ境界
- user_id / subject_id による厳格な分離（Professor側で強制）
- フロントエンドは物理制約を「信頼」して表示

### 外部API契約（SSOT）
- Professor: `docs/openapi.yaml`（`eduanimaR_Professor/docs/openapi.yaml` が正）
- 生成: Orval（`npm run api:generate`）
- 生成物: `src/shared/api/generated/`（コミット対象）
