# ERROR_CODES

## 目的
consumer（例: フロントエンド）が分岐できる安定したエラーコード体系を定義する。

## ルール
- `code` は機械可読な安定ID（破壊的変更を避ける）
- `message` は人間向け（内部情報は含めない）
- ドメイン別にプレフィックスを揃える（例: `USER_`, `ORDER_`）
- **フロントエンド側で `code` による分岐が必要** な場合、必ず OpenAPI の `responses` に明記する

## 共通レスポンス形式（確定）
```json
{
  "error": {
    "code": "USER_NOT_FOUND",
    "message": "User not found",
    "details": {
      "field": "user_id"
    },
    "request_id": "abc-123-def"
  }
}
```

## 標準エラーコード一覧
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

## フロントエンドとの同期
- Orval で生成された型には、OpenAPI で定義されたエラーレスポンスの型も含まれる
- フロントエンド側は生成された型で `error.code` を判定し、適切な UI を出す

---

## Phase 1 エラーコード（詳細）

Last-updated: 2026-02-17  
Status: Published  
Owner: @ttokunaga-ja

### フォーマット

すべてのエラーレスポンスは以下の形式で返す:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable message",
    "details": { /* optional */ },
    "request_id": "abc-123-def"
  }
}
```

**注意**: `request_id` は必須フィールドです（OpenAPIスキーマで定義）。

### エラーコード一覧（Phase 1）

| Code | HTTP Status | Message (EN) | Message (JA) | Frontend Action |
|------|-------------|--------------|--------------|-----------------|
| `FILE_TOO_LARGE` | 413 | File size exceeds 10MB limit | ファイルサイズが10MBを超えています | Show error toast |
| `INVALID_FILE_TYPE` | 400 | Unsupported file type. Allowed: PDF, PNG, JPG | 非対応のファイル形式です（PDF/PNG/JPG のみ） | Show error toast |
| `SUBJECT_NOT_FOUND` | 404 | Subject does not exist | 指定された科目が見つかりません | Redirect to subjects page |
| `FILE_NOT_FOUND` | 404 | File does not exist | 指定されたファイルが見つかりません | Show error message |
| `NO_SEARCH_RESULTS` | 200 | No relevant documents found for your question | 質問に関連する資料が見つかりませんでした | Show "no results" UI |
| `PROCESSING_TIMEOUT` | 504 | File processing timed out. Please try again later | ファイル処理がタイムアウトしました。しばらくしてから再度お試しください | Show retry button |
| `REQUEST_TIMEOUT` | 504 | Question processing timed out (60s limit) | 質問処理がタイムアウトしました（60秒制限） | Show error message |
| `INTERNAL_ERROR` | 500 | An unexpected error occurred. Please contact support | 予期しないエラーが発生しました。サポートにお問い合わせください | Show error page with support link |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests. Please wait before retrying | リクエストが多すぎます。しばらく待ってから再度お試しください | Show countdown timer |

### Frontend実装例（TypeScript）

```typescript
// src/shared/api/errors.ts
export const ERROR_MESSAGES: Record<string, { en: string; ja: string }> = {
  FILE_TOO_LARGE: {
    en: 'File size exceeds 10MB limit',
    ja: 'ファイルサイズが10MBを超えています',
  },
  INVALID_FILE_TYPE: {
    en: 'Unsupported file type. Allowed: PDF, PNG, JPG',
    ja: '非対応のファイル形式です（PDF/PNG/JPG のみ）',
  },
  SUBJECT_NOT_FOUND: {
    en: 'Subject does not exist',
    ja: '指定された科目が見つかりません',
  },
  FILE_NOT_FOUND: {
    en: 'File does not exist',
    ja: '指定されたファイルが見つかりません',
  },
  NO_SEARCH_RESULTS: {
    en: 'No relevant documents found for your question',
    ja: '質問に関連する資料が見つかりませんでした',
  },
  PROCESSING_TIMEOUT: {
    en: 'File processing timed out. Please try again later',
    ja: 'ファイル処理がタイムアウトしました。しばらくしてから再度お試しください',
  },
  REQUEST_TIMEOUT: {
    en: 'Question processing timed out (60s limit)',
    ja: '質問処理がタイムアウトしました（60秒制限）',
  },
  INTERNAL_ERROR: {
    en: 'An unexpected error occurred. Please contact support',
    ja: '予期しないエラーが発生しました。サポートにお問い合わせください',
  },
  RATE_LIMIT_EXCEEDED: {
    en: 'Too many requests. Please wait before retrying',
    ja: 'リクエストが多すぎます。しばらく待ってから再度お試しください',
  },
};

// エラーハンドリング例
export function handleApiError(error: ApiError, locale: 'en' | 'ja' = 'ja') {
  const message = ERROR_MESSAGES[error.code]?.[locale] || ERROR_MESSAGES.INTERNAL_ERROR[locale];
  
  switch (error.code) {
    case 'FILE_TOO_LARGE':
    case 'INVALID_FILE_TYPE':
      toast.error(message);
      break;
    
    case 'SUBJECT_NOT_FOUND':
      toast.error(message);
      router.push('/subjects');
      break;
    
    case 'NO_SEARCH_RESULTS':
      // 200番台なので特別扱い
      return { hasResults: false, message };
    
    case 'PROCESSING_TIMEOUT':
    case 'REQUEST_TIMEOUT':
      toast.error(message, { action: { label: 'Retry', onClick: retryFn } });
      break;
    
    case 'RATE_LIMIT_EXCEEDED':
      const retryAfter = error.details?.retry_after_seconds || 60;
      showCountdownTimer(retryAfter);
      break;
    
    default:
      toast.error(message);
      if (error.code === 'INTERNAL_ERROR') {
        reportToSentry(error);
      }
  }
}
```

### Backend実装例（Go）

```go
// internal/domain/errors/codes.go
package errors

type ErrorCode string

const (
    FileTooLarge         ErrorCode = "FILE_TOO_LARGE"
    InvalidFileType      ErrorCode = "INVALID_FILE_TYPE"
    SubjectNotFound      ErrorCode = "SUBJECT_NOT_FOUND"
    FileNotFound         ErrorCode = "FILE_NOT_FOUND"
    NoSearchResults      ErrorCode = "NO_SEARCH_RESULTS"
    ProcessingTimeout    ErrorCode = "PROCESSING_TIMEOUT"
    RequestTimeout       ErrorCode = "REQUEST_TIMEOUT"
    InternalError        ErrorCode = "INTERNAL_ERROR"
    RateLimitExceeded    ErrorCode = "RATE_LIMIT_EXCEEDED"
)

type ErrorResponse struct {
    Error struct {
        Code      ErrorCode              `json:"code"`
        Message   string                 `json:"message"`
        Details   map[string]interface{} `json:"details,omitempty"`
        RequestID string                 `json:"request_id"`
    } `json:"error"`
}

// エラー生成ヘルパー
func NewErrorResponse(code ErrorCode, message string, requestID string, details map[string]interface{}) *ErrorResponse {
    resp := &ErrorResponse{}
    resp.Error.Code = code
    resp.Error.Message = message
    resp.Error.RequestID = requestID
    resp.Error.Details = details
    return resp
}
```

### OpenAPI定義例

```yaml
components:
  schemas:
    ErrorResponse:
      type: object
      required:
        - error
      properties:
        error:
          type: object
          required:
            - code
            - message
            - request_id
          properties:
            code:
              type: string
              description: Stable application error code (see docs/03_integration/ERROR_CODES.md)
              enum:
                - FILE_TOO_LARGE
                - INVALID_FILE_TYPE
                - SUBJECT_NOT_FOUND
                - FILE_NOT_FOUND
                - NO_SEARCH_RESULTS
                - PROCESSING_TIMEOUT
                - REQUEST_TIMEOUT
                - INTERNAL_ERROR
                - RATE_LIMIT_EXCEEDED
            message:
              type: string
              description: User-facing message (no secrets / no internal details)
            details:
              type: object
              additionalProperties: true
            request_id:
              type: string

  responses:
    FileTooLarge:
      description: File size exceeds 10MB limit
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ErrorResponse'
          example:
            error:
              code: FILE_TOO_LARGE
              message: File size exceeds 10MB limit
              details:
                max_size_mb: 10
                actual_size_mb: 15.2
              request_id: req_abc123def456
```