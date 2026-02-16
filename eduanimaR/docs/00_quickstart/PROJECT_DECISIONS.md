---
Title: Project Decisions
Description: eduanimaRプロジェクトの技術決定事項とSSO設定のSSOT
Owner: @ttokunaga-ja
Status: Published
Last-updated: 2026-02-16
Tags: frontend, eduanimaR, project-decisions, authentication, api
---

# Project Decisions（SSOT）

Last-updated: 2026-02-16

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
  - **Librarian（Python）**: LangGraph Agent による検索戦略立案（ProfessorからHTTP/JSON経由で呼び出し）
  - **Professor → Librarian 通信プロトコル**: 
    - プロトコル: HTTP/JSON
    - エンドポイント: `POST /v1/librarian/search-agent`
    - Librarianはステートレス推論サービス（会話履歴・キャッシュ等の永続化なし）
    - Librarianへのフロントエンドからの直接通信は禁止（Professor経由のみ）

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
- **request_id管理**:
  - Professor APIリクエストに`X-Request-ID`ヘッダーを含める
  - Professor → Librarian推論ループでも`request_id`を伝播
  - SSEイベント・エラーレスポンスに`request_id`を含めて返却
  - フロントエンドはエラー報告時に`request_id`を含める

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
  - 具体的には「質問から関連箇所（資料名 + ページ番号）へ到達する時間」を計測
  - ユーザーが「理解する時間」を最大化するための指標

### ペルソナ要約
- **主要ペルソナ**: 忙しい学部生
  - 複数科目を並行履修し、資料が散在
  - 「どこに何が書いてあったか」を探す時間が負担
  - 資料の「着眼点」を示し、原典への回帰を促す支援が必要

### Professor / Librarian の責務境界
- **Professor（Go）**: データ所有者、DB/GCS/Kafka 直接アクセス、検索の物理実行、最終回答生成
  - DB/GCS/検索インデックスへの直接アクセス権限を持つ
  - 外向き API（HTTP/JSON + SSE）を提供
  - バッチ処理（OCR/Embedding）の実行管理
  - Librarianの`temp_index`を安定ID（`document_id`）に変換してフロントエンドへ返却
- **Librarian（Python）**: 検索戦略立案、停止判断、選定エビデンス（Librarianが選定した根拠箇所）の抽出（DB 直接アクセスなし）
  - LangGraph Agent による推論特化
  - Professor 経由でのみ検索実行を要求
  - 検索戦略の最適化と終了判定を担当
  - **Librarian推論ループパラメータ**:
    - `max_retries`: Librarian推論ループの上限回数（例: 5回）
    - Professorの物理検索パラメータ（取得件数k等）とは独立して管理
  - ステートレス設計: 会話履歴・キャッシュなし（1リクエストで推論完結）

### バックエンドサービス仕様への参照
- Professor サービス仕様: [`../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_PROFESSOR.md`](../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_PROFESSOR.md)
- Librarian サービス仕様: [`../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_LIBRARIAN.md`](../../eduanimaRHandbook/02_strategy/SERVICE_SPEC_EDUANIMA_LIBRARIAN.md)

### 用語の統一
- **Librarian推論ループ**: Librarianが検索戦略を立案し、Professor経由で検索を実行する反復プロセス
- **選定エビデンス**: Librarianが選定した根拠箇所（`selected_evidence`）
- **temp_index**: LLM可視の一時参照ID（Professorが安定ID `document_id` に変換してフロントエンドへ返却）

### 提供形態
- Chrome拡張機能（LMS利用中の介入）
- Webアプリケーション（復習用ダッシュボード）
- **導線統一**: どちらの導線でも同一のログイン体験（SSO/OAuth）と同一の権限境界を維持

### 認証・認可方針
- **Phase 1（ローカル開発）**: 認証スキップ（固定のdev-user使用）
- **Phase 2以降**: SSO認証実装（Google / Meta / Microsoft / LINE）
- **認可**: ユーザー別アクセス制限を厳格に実施（導線（拡張/WEB）に依存しない）

### ユーザー登録の境界（Phase 2 MUST）
- **新規登録**: Chrome拡張機能でのみ許可
- **Web版の役割**: 既存ユーザーの再ログイン・閲覧専用
- **未登録ユーザーのWeb版アクセス時**:
  - SSO認証は通過するが、Professor APIが `user_not_found` を返却
  - フロントエンドが拡張機能誘導画面を表示
  - 誘導先（優先順位順）:
    1. **Chrome Web Store**（公式配布）
    2. **GitHub Releases**（手動インストール用）
    3. **公式導入ガイド**（解説ブログ・ドキュメント）

### 誘導画面の実装要件（Phase 2）
| 項目 | 内容 |
|------|------|
| **ページパス** | `/auth/register-redirect` または `/onboarding/install-extension` |
| **表示条件** | Professor API `POST /auth/login` が `AUTH_USER_NOT_REGISTERED` を返した場合 |
| **UI要素** | タイトル、説明文、インストールボタン（Chrome Web Store）、補足リンク（GitHub、導入ガイド） |
| **アクセス制限** | 未認証ユーザーはSSO認証を要求、認証後に表示 |
| **デザイン** | MUI Pigment CSSでクリーンなオンボーディングUI |
| **トラッキング** | 誘導画面の表示回数、各リンクのクリック数を記録（Professor経由） |

### 拡張機能配布URL（Phase 2で確定）
実装時に `src/shared/config/extension-urls.ts` で以下を管理:
```typescript
export const EXTENSION_URLS = {
  chromeWebStore: 'https://chrome.google.com/webstore/detail/[extension-id]',
  githubReleases: 'https://github.com/[org]/[repo]/releases',
  officialGuide: '[公式導入ガイドURL]',
} as const;
```

### Professor API仕様（Phase 2で実装）
未登録ユーザーの応答例:
```json
{
  "error": {
    "code": "AUTH_USER_NOT_REGISTERED",
    "message": "User is authenticated but not registered. Please install the Chrome extension to register.",
    "extension_urls": {
      "chrome_web_store": "https://chrome.google.com/webstore/detail/[extension-id]",
      "github_releases": "https://github.com/[org]/[repo]/releases",
      "official_guide": "[公式導入ガイドURL]"
    }
  }
}
```

### バックエンド境界
- **Professor（Go）**: データの守護者、APIのSSOT（OpenAPI）、唯一DBに直接アクセス
  - Librarian推論ループを`POST /v1/librarian/search-agent`で呼び出し
  - Librarianの`selected_evidence`（`temp_index`配列）を受け取り、安定ID（`document_id`）に変換
- **Librarian（Python）**: Librarian推論ループ専門サービス（ステートレス、Professorのみが呼ぶ）
  - 会話履歴・キャッシュなし（1リクエスト内で推論完結）
  - `max_retries`上限で推論ループを制御
- **フロントエンド**: Professor の OpenAPI（HTTP/JSON + SSE）のみを呼ぶ
  - Librarianへの直接通信は禁止

### データ境界・プライバシー
- ユーザー別データ分離がデフォルト
- 共有範囲: 将来「科目の資料セット」のみ共有、質問履歴や学習ログは共有しない

### ロードマップ（Phase 1〜4）
- **Phase 1**: ローカル開発、基本的なQ&A機能、資料管理
- **Phase 2**: SSO認証、本番環境デプロイ
- **Phase 3**: Librarian推論ループ連携、高度な検索
  - **Librarian連携UI要件**:
    - 検索ループ進行表示（`widgets/search-loop-status`）
    - 選定エビデンス表示（`entities/evidence`）
    - 推論理由の可視化（なぜこのエビデンスが選ばれたか）
  - Professor SSEでのリアルタイム配信（`search_loop_progress`、`evidence_selected`イベント）
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
