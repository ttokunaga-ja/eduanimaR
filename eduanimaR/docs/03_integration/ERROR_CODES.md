---
Title: Error Codes
Description: eduanimaR共通エラーコード一覧とUI表示マッピング
Owner: @ttokunaga-ja
Status: Published
Last-updated: 2026-02-16
Tags: frontend, eduanimaR, error-codes, professor, api
---

# Error Codes（エラーコード一覧）

Last-updated: 2026-02-16

本ドキュメントは Professor（Go）の `ERROR_CODES.md` と同期します。

## 目的
- バックエンド（Professor / Librarian）とフロントエンドで統一のエラーコード体系を維持
- エラー発生時の UI 表示・ユーザー誘導を標準化
- 監視・デバッグ時のエラー分類を容易にする

## エラーコード体系
- **1xxx**: 認証・認可エラー
- **2xxx**: リクエストエラー（バリデーション、必須パラメータ不足）
- **3xxx**: リソース不在エラー（404相当）
- **4xxx**: 外部サービスエラー（Gemini API、GCS、Kafka）
- **5xxx**: 内部エラー（予期しないエラー）

---

## 主要エラーコード

| コード | 名称 | 説明 | UI表示 | リトライ |
|:---|:---|:---|:---|:---|
| `1001` | `AUTH_REQUIRED` | 認証が必要 | 「ログインしてください」ダイアログ表示、ログイン画面へ遷移 | 不可 |
| `1002` | `AUTH_INVALID_TOKEN` | トークン無効 | セッションをクリアし、再ログイン促進 | 不可 |
| `1003` | `AUTH_FORBIDDEN` | 権限不足 | 「この操作は許可されていません」トースト表示 | 不可 |
| `2001` | `VALIDATION_FAILED` | バリデーションエラー | フォームのフィールドごとにエラーメッセージ表示 | 不可 |
| `2002` | `FILE_TOO_LARGE` | ファイルサイズ超過 | 「ファイルサイズは最大50MBです」トースト表示 | 不可 |
| `2003` | `INVALID_FILE_TYPE` | 非対応ファイル形式 | 「対応形式: PDF, DOCX, PPTX, 画像」トースト表示 | 不可 |
| `3001` | `SUBJECT_NOT_FOUND` | 科目が存在しない | 「科目が見つかりません」空状態表示 | 不可 |
| `3002` | `FILE_NOT_FOUND` | ファイルが存在しない | 「ファイルが削除されたか、アクセス権限がありません」 | 不可 |
| `4001` | `GEMINI_API_ERROR` | Gemini API エラー | 「AI処理が一時的に利用できません。しばらく待ってから再試行してください」 | 可 |
| `4002` | `GCS_ERROR` | GCS エラー | 「ファイルの取得に失敗しました」 | 可 |
| `4003` | `KAFKA_ERROR` | Kafka エラー | 「処理の開始に失敗しました。再試行してください」 | 可 |
| `5001` | `INTERNAL_ERROR` | 内部エラー | 「予期しないエラーが発生しました。サポートに連絡してください」 | 可（最大3回） |

---

## 認証関連エラー（Phase 2）

### AUTH_USER_NOT_REGISTERED
- **コード**: `AUTH_USER_NOT_REGISTERED`
- **使用Phase**: Phase 2で使用
- **発生条件**: SSO認証は成功したが、Professorにユーザーレコードが存在しない
- **ステータスコード**: `403 Forbidden`
- **UI挙動**: 拡張機能誘導画面（`/auth/register-redirect`）へ遷移
- **表示メッセージ**: 「eduanimaRをご利用いただくには、Chrome拡張機能のインストールが必要です」
- **アクション**: Chrome Web Store、GitHub、導入ガイドへのリンクを表示
- **Professor API**: `POST /auth/login`が返却
- **ProfessorレスポンスとFrontendの処理**:
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

### AUTH_EXTENSION_REQUIRED
- **発生条件**: Web版で新規登録フォームへのアクセスを検知
- **ステータスコード**: `403 Forbidden`
- **UI挙動**: 同上
- **表示メッセージ**: 「新規登録は拡張機能でのみ可能です」

---

## フロントエンド実装方針

### 1. エラーコードの受け取り
Professor / Librarian からのエラーレスポンスは以下の形式：

```json
{
  "code": "AUTH_REQUIRED",
  "message": "Authentication required",
  "details": {}
}
```

### 2. UI メッセージへの変換
`src/shared/api/errors.ts` にエラーコードマップを定義：

```typescript
export const ERROR_MESSAGES: Record<string, string> = {
  AUTH_REQUIRED: 'ログインしてください',
  AUTH_INVALID_TOKEN: 'セッションが無効です。再ログインしてください',
  AUTH_FORBIDDEN: 'この操作は許可されていません',
  VALIDATION_FAILED: '入力内容を確認してください',
  FILE_TOO_LARGE: 'ファイルサイズは最大50MBです',
  INVALID_FILE_TYPE: '対応形式: PDF, DOCX, PPTX, 画像',
  SUBJECT_NOT_FOUND: '科目が見つかりません',
  FILE_NOT_FOUND: 'ファイルが削除されたか、アクセス権限がありません',
  GEMINI_API_ERROR: 'AI処理が一時的に利用できません。しばらく待ってから再試行してください',
  GCS_ERROR: 'ファイルの取得に失敗しました',
  KAFKA_ERROR: '処理の開始に失敗しました。再試行してください',
  INTERNAL_ERROR: '予期しないエラーが発生しました。サポートに連絡してください',
};
```

### 3. リトライ戦略
リトライ可能なエラー（`4xxx`, `5001`）は以下の戦略を採用：

- **初回リトライ**: 1秒後
- **2回目リトライ**: 2秒後
- **3回目リトライ**: 4秒後（最大3回）
- **指数バックオフ**: `Math.min(1000 * Math.pow(2, retryCount), 10000)`

### 4. エラーバウンダリ
予期しないエラー（`5001` 等）は React Error Boundary でキャッチし、フォールバック UI を表示：

```typescript
<ErrorBoundary
  fallback={<ErrorFallback />}
  onError={(error, errorInfo) => {
    // エラーログを送信
    logError(error, errorInfo);
  }}
>
  <App />
</ErrorBoundary>
```

### 5. SSE エラーの扱い
SSE（`/qa/stream`）で受信する `error` イベントは以下の形式：

```json
{
  "type": "error",
  "code": "GEMINI_API_ERROR",
  "message": "Gemini API error occurred"
}
```

クライアント側の実装：

```typescript
eventSource.addEventListener('error', (event) => {
  const error = JSON.parse(event.data);
  const userMessage = ERROR_MESSAGES[error.code] || 'エラーが発生しました';
  showToast(userMessage, 'error');
  
  // リトライ可能なエラーの場合は再接続を試みる
  if (isRetryable(error.code)) {
    scheduleRetry();
  }
});
```

---

## エラーコードの追加・更新手順

1. **Professor（Go）で追加**:
   - Professor の `ERROR_CODES.md` に新しいエラーコードを追加
   - Go コード内で該当エラーを返す実装を追加

2. **フロントエンドに同期**:
   - 本ドキュメントにエラーコードを追記
   - `src/shared/api/errors.ts` に UI メッセージを追加
   - 必要に応じて UI コンポーネントを更新

3. **CI で差分検出**:
   - Professor の `ERROR_CODES.md` とフロントエンドの本ドキュメントを比較
   - 差分がある場合は CI を失敗させる（同期忘れ防止）

---

## 禁止事項

- `message` 文字列で分岐しない（必ず `code` を使用）
- エラーコードをハードコーディングしない（`ERROR_MESSAGES` マップを使用）
- Professor のエラーコード体系と異なる独自のコードを定義しない

---

## eduanimaR共通エラーコード

### Professor API エラーコード

| コード | HTTPステータス | ユーザー向けメッセージ |
|--------|---------------|----------------------|
| `MATERIAL_NOT_FOUND` | 404 | 資料が見つかりませんでした |
| `SUBJECT_ACCESS_DENIED` | 403 | この科目へのアクセス権限がありません |
| `SEARCH_TIMEOUT` | 504 | 検索がタイムアウトしました。もう一度お試しください |
| `REASONING_FAILED` | 500 | 回答生成に失敗しました |
| `INVALID_QUESTION` | 400 | 質問の形式が正しくありません |
| `RATE_LIMIT_EXCEEDED` | 429 | リクエストが多すぎます。しばらく待ってから再試行してください |
| `AUTHENTICATION_REQUIRED` | 401 | ログインが必要です |
| `TOKEN_EXPIRED` | 401 | セッションの有効期限が切れました。再度ログインしてください |

### フロントエンド実装

```typescript
// src/shared/api/error-codes.ts
export const ERROR_CODES: Record<string, string> = {
  MATERIAL_NOT_FOUND: '資料が見つかりませんでした',
  SUBJECT_ACCESS_DENIED: 'この科目へのアクセス権限がありません',
  SEARCH_TIMEOUT: '検索がタイムアウトしました。もう一度お試しください',
  REASONING_FAILED: '回答生成に失敗しました',
  INVALID_QUESTION: '質問の形式が正しくありません',
  RATE_LIMIT_EXCEEDED: 'リクエストが多すぎます。しばらく待ってから再試行してください',
  AUTHENTICATION_REQUIRED: 'ログインが必要です',
  TOKEN_EXPIRED: 'セッションの有効期限が切れました。再度ログインしてください',
};

export function getErrorMessage(code: string): string {
  return ERROR_CODES[code] || 'エラーが発生しました';
}
```

### バックエンドとの同期
- Professor側の`ERROR_CODES.md`と定期的に同期
- 新規エラーコード追加時はフロントエンド側も更新

---

## Librarian由来エラー（Phase 3以降）

### LIBRARIAN_LOOP_LIMIT
- **コード**: `LIBRARIAN_LOOP_LIMIT`
- **発生条件**: Librarian推論ループが`max_retries`上限に達した
- **ステータスコード**: `200 OK`（部分的な結果を返却）
- **UI挙動**: 
  - 「検索ループが上限に達しました。部分的な結果を表示します」警告トースト表示
  - 取得できた選定エビデンスを表示
- **リトライ**: 不可（再質問を促す）
- **フロントエンドの推奨対応**: 
  - 部分結果を表示しつつ、ユーザーに質問の具体化を促す
  - 「より具体的な質問で再試行してください」メッセージを表示

### LIBRARIAN_MODEL_FAILURE
- **コード**: `LIBRARIAN_MODEL_FAILURE`
- **発生条件**: Gemini API呼び出しエラー（Librarian内部）
- **ステータスコード**: `500 Internal Server Error`
- **UI挙動**: 
  - 「AI処理が一時的に利用できません。しばらく待ってから再試行してください」エラートースト表示
  - Professor側でフォールバック処理を実行（基本検索結果を返却）
- **リトライ**: 可（指数バックオフ）
- **フロントエンドの推奨対応**: 
  - 自動リトライ（最大3回、1秒 → 2秒 → 4秒）
  - リトライ後も失敗する場合、ユーザーに通知

### LIBRARIAN_TIMEOUT
- **コード**: `LIBRARIAN_TIMEOUT`
- **発生条件**: Librarian推論時間上限超過（例: 30秒）
- **ステータスコード**: `504 Gateway Timeout`
- **UI挙動**: 
  - 「推論がタイムアウトしました。質問を簡略化して再試行してください」エラートースト表示
- **リトライ**: 可（ユーザーによる手動リトライ）
- **フロントエンドの推奨対応**: 
  - ユーザーに質問の簡略化を促す
  - タイムアウト履歴を記録（分析用）

---

## エラーコードとProfessor APIレスポンスの対応表

| エラーコード | HTTPステータス | リトライ可否 | ユーザー通知文言 | フロントエンド推奨対応 |
|:---|:---|:---|:---|:---|
| `LIBRARIAN_LOOP_LIMIT` | 200 OK | 不可 | 「検索ループが上限に達しました。部分的な結果を表示します」 | 部分結果表示 + 質問具体化を促す |
| `LIBRARIAN_MODEL_FAILURE` | 500 | 可 | 「AI処理が一時的に利用できません。しばらく待ってから再試行してください」 | 自動リトライ（指数バックオフ） |
| `LIBRARIAN_TIMEOUT` | 504 | 可（手動） | 「推論がタイムアウトしました。質問を簡略化して再試行してください」 | 質問簡略化を促す + タイムアウト履歴記録 |
| `AUTH_USER_NOT_REGISTERED` | 403 | 不可 | 「eduanimaRをご利用いただくには、Chrome拡張機能のインストールが必要です」 | 拡張機能誘導画面へ遷移 |
| `GEMINI_API_ERROR` | 500 | 可 | 「AI処理が一時的に利用できません。しばらく待ってから再試行してください」 | 自動リトライ（指数バックオフ） |
| `INTERNAL_ERROR` | 500 | 可 | 「予期しないエラーが発生しました。サポートに連絡してください」 | 自動リトライ（最大3回） + エラー報告 |
