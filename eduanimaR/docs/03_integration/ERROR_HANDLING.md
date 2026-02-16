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

## サービスミッション（North Star）

**Mission**: 学習者が、配布資料や講義情報の中から「今見るべき場所」と「次に取るべき行動」を素早く特定できるようにし、理解と継続を支援する

**North Star Metric**: 資料から根拠箇所に到達するまでの時間短縮

**参照**: [`../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md`](../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md)

## 関連ドキュメント
- **Handbook品質原則**: [`../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md`](../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md)
- **Professor エラーコード**: [`../../eduanimaR_Professor/docs/03_integration/ERROR_CODES.md`](../../eduanimaR_Professor/docs/03_integration/ERROR_CODES.md)
- **Professor API仕様**: [`../../eduanimaR_Professor/docs/README.md`](../../eduanimaR_Professor/docs/README.md)
- **フロントエンドエラーコード**: `ERROR_CODES.md`
- **観測性**: `../05_operations/OBSERVABILITY.md`

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

## Professor APIエラーレスポンス形式（HTTP/JSON + SSE）

### Professor/Librarian責務境界

#### Professor（Go）の責務
- **データ守護者（唯一の権限者）**: DB/GCS/Kafka直接アクセス権限を持つ
- **Phase 2（大戦略）**: タスク分割・停止条件決定
- **Phase 3（物理実行）**: ハイブリッド検索(RRF統合)、動的k値設定、権限強制
- **Phase 4（合成）**: Gemini 3 Proで最終回答生成
- **外向きAPI提供**: HTTP/JSON + SSEでフロントエンドと通信
- **エラーレスポンス**: 統一されたエラーコードとメッセージ

#### Librarian（Python）の責務
- **Phase 3（小戦略）**: LangGraphによるLibrarian推論ループ（最大5回推奨）
- **ステートレス**: 会話履歴・キャッシュなし
- **DB直接アクセス禁止**: Professor経由でのみ検索実行
- **通信**: **gRPC（双方向ストリーミング）** でProfessorと通信
- **エラー伝播**: LibrarianエラーはProfessor経由でフロントエンドへ

#### Frontend責務
- **ProfessorのHTTP/JSON+SSEのみ**: Librarian直接通信禁止
- **エラーハンドリング**: Professor APIエラーレスポンスを統一的に処理
- **request_id伝播**: すべてのエラーで`request_id`を追跡

参照:
- [`../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md`](../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md)
- [`../../eduanimaR_Professor/docs/README.md`](../../eduanimaR_Professor/docs/README.md)

### エラーレスポンス形式

```json
{
  "code": "MATERIAL_NOT_FOUND",
  "message": "Material with ID 'abc123' not found",
  "request_id": "req-1234567890",
  "details": {
    "materialId": "abc123"
  }
}
```

**重要**: すべてのエラーレスポンスに`request_id`を含め、追跡可能性を確保します（Handbook品質原則準拠）。

## フロントエンド統一エラーハンドリング

### TanStack Query onError（request_id伝播）

```typescript
interface ErrorResponse {
  code: string;
  message: string;
  request_id: string; // 追跡可能性（Handbook品質原則準拠）
  details?: Record<string, unknown>;
}

const { data, error } = useQuery({
  queryKey: ['materials', id],
  queryFn: () => getMaterial(id),
  onError: (error: AxiosError<ErrorResponse>) => {
    const code = error.response?.data?.code;
    const message = getErrorMessage(code);
    const requestId = error.response?.data?.request_id;
    
    // request_idを含めてエラーログに記録（追跡可能性確保）
    console.error(`[${requestId}] Error: ${code} - ${message}`);
    showErrorToast(message, { requestId });
  },
});
```

### Axios Interceptor（request_id伝播と説明可能性）

```typescript
apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError<ErrorResponse>) => {
    const code = error.response?.data?.code;
    const requestId = error.response?.data?.request_id;
    
    // request_idをログに記録（追跡可能性）
    console.error(`[${requestId}] API Error: ${code}`);
    
    // 認証エラー
    if (code === 'TOKEN_EXPIRED') {
      // リフレッシュトークンで再試行
      return refreshAndRetry(error.config);
    }
    
    // Librarianタイムアウトエラー（Librarian推論ループ失敗）
    if (code === 'LIBRARIAN_TIMEOUT') {
      showErrorToast('Librarian推論ループがタイムアウトしました。再試行してください。', { requestId });
    }
    
    return Promise.reject(error);
  }
);
```

## SSE接続エラー処理（Professor HTTP/JSON + SSE）

### Professor の SSE ストリーミング中のエラーハンドリング

#### Professor/Librarian責務とSSEエラー
- **Professor**: HTTP/JSON + SSEでフロントエンドと通信、Librarian推論ループのエラーもSSEで伝播
- **Librarian**: **gRPC（双方向ストリーミング）** でProfessorと通信、ステートレス推論ループ（最大5回推奨）
- **Frontend**: ProfessorのSSEのみ受信、Librarian直接通信禁止

#### SSE 切断時の再接続戦略（透明性確保）

**指数バックオフ再接続**:
- 初回: 1秒後に再接続
- 2回目: 2秒後に再接続
- 3回目: 4秒後に再接続
- 4回目: 8秒後に再接続
- 5回目: 16秒後に再接続
- 最大リトライ回数: 5回（Librarian推論ループと同期）
- リトライ上限到達時: ユーザーへエラー通知（request_id含む）

```typescript
function connectSSE(url: string, maxRetries = 5) {
  let retries = 0;
  let requestId: string | null = null;
  
  function connect() {
    const eventSource = new EventSource(url);
    
    // request_idを取得（追跡可能性）
    eventSource.addEventListener('open', () => {
      // 初回接続時にrequest_idを取得
      fetch(url, { method: 'HEAD' }).then((res) => {
        requestId = res.headers.get('X-Request-ID');
      });
    });
    
    eventSource.onerror = () => {
      eventSource.close();
      
      if (retries < maxRetries) {
        const delay = Math.pow(2, retries) * 1000; // 1s, 2s, 4s, 8s, 16s
        console.warn(`[${requestId}] SSE reconnecting (${retries + 1}/${maxRetries})...`);
        setTimeout(() => {
          retries++;
          connect();
        }, delay);
      } else {
        // 追跡可能性: request_idを含めてエラー通知
        showErrorToast('接続に失敗しました', { requestId });
        console.error(`[${requestId}] SSE connection failed after ${maxRetries} retries`);
      }
    };
    
    return eventSource;
  }
  
  return connect();
}
```

#### ストリーミング中のエラーイベント処理（説明可能性・透明性）

**エラーイベントの種類**:
- `error`: 一般的なエラー（リトライ可能）
- `rate_limit`: レート制限超過（待機後リトライ）
- `internal_error`: サーバー内部エラー（リトライ制限あり）
- `timeout`: タイムアウト（再送信可能）
- `LIBRARIAN_TIMEOUT`: Librarian推論ループタイムアウト（Professor経由で通知）

**処理方針（Handbook品質原則準拠）**:
```typescript
eventSource.addEventListener('error', (event) => {
  const errorData = JSON.parse(event.data);
  const errorCode = errorData.code;
  const requestId = errorData.request_id; // 追跡可能性
  const message = errorData.message; // 説明可能性
  
  // request_idをログに記録（追跡可能性）
  console.error(`[${requestId}] SSE Error: ${errorCode} - ${message}`);
  
  switch (errorCode) {
    case 'RATE_LIMIT_EXCEEDED':
      // レート制限: 待機時間を表示してリトライ（透明性）
      const retryAfter = errorData.retryAfter || 60;
      showWarningToast(`レート制限に達しました。${retryAfter}秒後に再試行します。`, { requestId });
      setTimeout(() => reconnect(), retryAfter * 1000);
      break;
      
    case 'INTERNAL_ERROR':
      // 内部エラー: ユーザーに通知して終了（説明可能性）
      showErrorToast(`サーバーエラーが発生しました。後ほど再試行してください。（ID: ${requestId}）`);
      eventSource.close();
      break;
      
    case 'TIMEOUT':
      // タイムアウト: 自動リトライ（透明性）
      showWarningToast(`タイムアウトしました。再接続中...（ID: ${requestId}）`);
      reconnect();
      break;
      
    case 'LIBRARIAN_TIMEOUT':
      // Librarian推論ループタイムアウト（Professor経由で通知）
      showErrorToast(`Librarian推論ループがタイムアウトしました。再試行してください。（ID: ${requestId}）`);
      eventSource.close();
      break;
      
    default:
      // その他のエラー: 詳細をログに記録してユーザーに通知
      console.error(`[${requestId}] Unexpected SSE Error:`, errorData);
      showErrorToast(`エラーが発生しました: ${message}（ID: ${requestId}）`);
  }
});
```

#### エラーコードと Handbook 品質原則の整合性

Professor の HTTP/JSON + SSE エラーレスポンス形式とフロントエンドのエラーハンドリングは、Handbook で定義された品質原則に従います:

| 品質原則 | 実装方法 | 参照 |
|---------|---------|------|
| **追跡可能性** | すべてのエラーに `request_id` を含め、SSEで伝播 | [`../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md`](../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md) |
| **説明可能性** | エラーコード(`code`)とメッセージ(`message`)を分離し、ユーザー理解可能な説明を提供 | [`../../eduanimaR_Professor/docs/03_integration/ERROR_CODES.md`](../../eduanimaR_Professor/docs/03_integration/ERROR_CODES.md) |
| **透明性** | SSEでリアルタイムにエラー状態を表示（Librarian推論ループのエラー含む） | 本ドキュメント |

### Professor/Librarian責務境界とエラー処理

#### Professor（Go）のエラー処理責務
- DB/GCS/Kafka直接アクセスエラーの捕捉
- Librarianエラーの受信とフロントエンドへの伝播（HTTP/JSON経由）
- SSEでのリアルタイムエラー通知
- request_id伝播の保証

#### Librarian（Python）のエラー処理責務
- Librarian推論ループのエラー検出（最大5回試行）
- Professor経由でのエラー通知（**gRPC**）
- DB直接アクセス禁止エラーの検出

#### Frontend エラー処理責務
- Professor SSEエラーイベントの受信
- request_id伝播による追跡可能性確保
- ユーザー向けエラー表示（説明可能性・透明性）
- Librarian直接通信禁止の徹底

**参照**:
- **Handbook品質原則**: [`../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md`](../../eduanimaRHandbook/01_philosophy/MISSION_VALUES.md)
- **Professor エラーコード**: [`../../eduanimaR_Professor/docs/03_integration/ERROR_CODES.md`](../../eduanimaR_Professor/docs/03_integration/ERROR_CODES.md)
- **Professor MICROSERVICES_MAP**: [`../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md`](../../eduanimaR_Professor/docs/01_architecture/MICROSERVICES_MAP.md)
- **Librarian README**: [`../../eduanimaR_Librarian/docs/README.md`](../../eduanimaR_Librarian/docs/README.md)
- **フロントエンドエラーコード**: `ERROR_CODES.md`

---

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
