---
Title: Error Handling
Description: eduanimaRのエラーハンドリング統一方針
Owner: @ttokunaga-ja
Status: Published
Last-updated: 2026-02-15
Tags: frontend, eduanimaR, error-handling, professor, api
---

# Error Handling（RSC / Route Handler / Client）

Last-updated: 2026-02-15

このドキュメントは、フロントエンドにおける「失敗の扱い」を統一し、
- エラーの握りつぶし
- 画面ごとのバラバラな例外処理
- 重要障害を運用で検知できない
を防ぐための契約です。

関連：
- エラーコード：`ERROR_CODES.md`
- 観測性：`../05_operations/OBSERVABILITY.md`

---

## 結論（Must）

- エラーは **分類** して扱う（ユーザー操作/権限/入力/一時障害/致命）
- “ユーザーに見せる失敗” は **UIパターンを固定**（inline/form/toast/page）
- “運用で検知すべき失敗” は **必ず観測に載せる**（握りつぶさない）
- 境界（RSC / Route Handler / Client）ごとに責務を分ける
- ユーザーに表示する文言は**すべて翻訳キー（変数）で管理**し、表示内容は各言語の JSON ファイルから読み出すこと。表示用のマッピングや未翻訳・未使用キーの検出は CI でチェックする仕組みを整備する

---

## 1) エラー分類（推奨）

- Validation：入力が不正（フォームに反映）
- AuthN/AuthZ：未ログイン/権限不足（ログイン導線 or 403）
- Not Found：対象が存在しない（404ページ or 空状態）
- Conflict：同時更新/状態不整合（再読み込み/再試行）
- Rate Limit：待って再試行（リトライ間隔をUIで制御）
- Upstream Timeout/Unavailable：一時障害（再試行/フォールバック）
- Internal：想定外（error boundary + 監視）

具体のコード体系は `ERROR_CODES.md` の表をSSOTとする。

---

## 2) どこで扱うか（境界ごとの責務）

### RSC（Server Component）
- 目的：初期表示の失敗を「ページ単位」で扱う
- 原則：
  - recover できないなら throw して route error boundary に寄せる
  - recover できるなら「空状態/フォールバック」を返す（ただし観測は残す）

### Route Handler / Server Action
- 目的：HTTP境界としてエラーを“契約化”して返す
- 原則：
  - 4xx/5xx の区別を崩さない
  - UI が必要とする最小情報（code/message/requestId）を返す

### Client Component
- 目的：ユーザー操作に対する失敗を、UXとして一貫して扱う
- 原則：
  - フォームは inline（フィールド/フォーム上部）
  - グローバルな失敗は toast（ただし連打で荒らさない）
  - 致命は error boundary

---

## 3) UIパターン（固定）

- Form validation：フィールドエラー/フォームエラー
- Permission：ログイン導線 or アクセス不可表示
- Not Found：ページの Not Found（ルートの責務）
- Retryable（timeout/unavailable）：再試行ボタン + 状態保持
- Unexpected：error boundary（ユーザー向け文言は固定）

---

## 4) 禁止（AI/人間共通）

- catchして `console.error` だけで終わる（運用に乗らない）
- エラーコードの “文字列比較の散乱”（一箇所で分類する）
- 画面ごとに勝手な文言/扱いを定義する

---

## Professor APIエラーレスポンス形式

```json
{
  "code": "MATERIAL_NOT_FOUND",
  "message": "Material with ID 'abc123' not found",
  "details": {
    "materialId": "abc123"
  }
}
```

## フロントエンド統一エラーハンドリング

### TanStack Query onError

```typescript
const { data, error } = useQuery({
  queryKey: ['materials', id],
  queryFn: () => getMaterial(id),
  onError: (error: AxiosError<ErrorResponse>) => {
    const code = error.response?.data?.code;
    const message = getErrorMessage(code);
    showErrorToast(message);
  },
});
```

### Axios Interceptor

```typescript
apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError<ErrorResponse>) => {
    const code = error.response?.data?.code;
    
    // 認証エラー
    if (code === 'TOKEN_EXPIRED') {
      // リフレッシュトークンで再試行
      return refreshAndRetry(error.config);
    }
    
    return Promise.reject(error);
  }
);
```

## SSE接続エラー処理

### 指数バックオフ再接続

```typescript
function connectSSE(url: string, maxRetries = 5) {
  let retries = 0;
  
  function connect() {
    const eventSource = new EventSource(url);
    
    eventSource.onerror = () => {
      eventSource.close();
      
      if (retries < maxRetries) {
        const delay = Math.pow(2, retries) * 1000; // 1s, 2s, 4s, 8s, 16s
        setTimeout(() => {
          retries++;
          connect();
        }, delay);
      } else {
        showErrorToast('接続に失敗しました');
      }
    };
    
    return eventSource;
  }
  
  return connect();
}
```

## エラー表示UI

### トースト通知
```typescript
import { toast } from 'react-hot-toast';

showErrorToast('資料が見つかりませんでした');
```

### エラーページ
```typescript
// app/error.tsx
export default function Error({ error }: { error: Error }) {
  return (
    <div>
      <h1>エラーが発生しました</h1>
      <p>{error.message}</p>
    </div>
  );
}
```

### インラインエラー
```typescript
{error && <p className="text-red-500">{error.message}</p>}
```
