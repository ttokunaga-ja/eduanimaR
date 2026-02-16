---
Title: Data Access Layer
Description: eduanimaRのデータアクセス層ポリシーとProfessor API統合
Owner: @ttokunaga-ja
Status: Published
Last-updated: 2026-02-16
Tags: frontend, eduanimaR, dal, api, professor
---

# Data Access Layer（DAL）ポリシー

Last-updated: 2026-02-16

このドキュメントは、Next.js App Router（RSC）時代の「データ取得の置き場所」を固定し、
- 認可漏れ
- 秘匿情報の露出
- “どこでも何でも取得する” ことによる複雑性
を防ぐための **運用契約** です。

本テンプレートの前提スタックは [STACK.md](../02_tech_stack/STACK.md) を参照。

関連ドキュメント:
- API契約: `../03_integration/API_CONTRACT_WORKFLOW.md`
- 検索パラメータ: `../00_quickstart/PROJECT_DECISIONS.md`（検索パラメータの決定事項）

---

## 結論（Must）

- **Server Component（RSC）でのデータ取得は、原則として DAL 経由で行う**
- **DAL は server-only**（クライアントに import されるとビルドが落ちる状態を目指す）
- **DAL は認可チェックと DTO 最小化を責務に含める**
- **`process.env` / secret の参照は DAL に集約**（他レイヤーに散らさない）

### eduanimaR固有の考慮事項
- **Professor OpenAPI統合**: Orvalで生成されたクライアントを使用
- **SSE接続**: `/v1/qa/ask` はSSEのため、`EventSource`または`fetch`（ReadableStream）で実装
- **Librarian推論状態**: SSEイベント（`thinking`, `searching`, `evidence`）をUI状態に反映
- **認証**: Phase 1はdev-user固定、Phase 2でSSO統合

---

## なぜ DAL が必要か（2026 / RSC 前提）

RSC により「サーバで自由に取得できる」ようになった反面、以下の事故が起きやすくなります。

- **“つい” Server Component から DB/API を直叩きしてしまい、認可が抜ける**
- Server → Client への props で **過剰なデータ（秘匿・個人情報）を渡してしまう**
- 取得箇所が散って **キャッシュ/再検証の設計が破綻** する

DAL を 1 箇所に寄せることで、監査・レビュー・変更が容易になります。

---

## 置き場所（推奨）

本テンプレートでは、API クライアント生成（Orval）を `src/shared/api/generated` に置く前提のため、
DAL は「生成物の上に薄い手書きレイヤー」として分離します。

推奨パターン（例）：

```text
src/shared/api/
├── generated/                  # 自動生成（手編集禁止）
├── client.ts                   # 共通設定（baseURL/認証/共通fetcher）
├── errors.ts                   # エラー分類
├── dal/                        # DAL（server-only）
│   ├── user.ts                 # getCurrentUserDTO 等
│   └── product.ts
└── index.ts                    # Public API
```

DAL の modules は先頭に `import 'server-only'` を置く運用を推奨します（依存して良いのは server 側のみ）。

---

## DAL の責務（Must）

### 1) 認可（Authorization）
- **毎回** “現在のユーザーがそのデータにアクセスしてよいか” をチェックする
- クライアントから来る入力（params/searchParams/formData など）は **信用しない**（必ず再検証）

### 2) DTO 最小化（Data Minimization）
- Client Component に渡すデータは **表示に必要な最小フィールドのみ**
- “バックエンドのレスポンスをそのまま props に渡す” を禁止

### 3) キャッシュ契約（CACHING_STRATEGY.md と整合）
- DAL は `fetch` のキャッシュ方針（`next.revalidate` / `next.tags` 等）を統一する起点
- invalidate は Server Action/Route Handler 側で明示する

関連：キャッシュの運用契約は [CACHING_STRATEGY.md](./CACHING_STRATEGY.md)

---

## 禁止（AI/人間共通）

- RSC が Route Handler を呼ぶ（サーバ内で **余計な HTTP hop** を作る）
- Server Component から取得した “生データ” を Client に丸渡しする
- secrets / `process.env` を DAL 以外で参照する（`NEXT_PUBLIC_` 以外）

---

## 監査チェック（最低限）

- `"use client"` ファイルが server-only module を import していないか
- Client props の型が過剰に広くないか（バックエンドの型をそのまま使っていないか）
- DAL が “認可” と “DTO 最小化” を必ずしているか

---

## Professor API統一呼び出し

### Orval生成クライアントの配置
- **生成先**: `src/shared/api/generated/`
- **生成コマンド**: `npm run api:generate`
- **SSOT**: `eduanimaR_Professor/docs/openapi.yaml`

### Phase別の実装切り替え

#### Phase 1: 認証スキップ
```typescript
// src/shared/api/client.ts
export const apiClient = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_BASE_URL, // http://localhost:8080
  headers: {
    'X-Dev-User': 'dev-user', // Phase 1のみ
  },
});
```

#### Phase 2: トークン付与
```typescript
// src/shared/api/client.ts
export const apiClient = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_BASE_URL, // https://professor.example.com
});

apiClient.interceptors.request.use(async (config) => {
  const session = await getSession();
  if (session?.accessToken) {
    config.headers.Authorization = `Bearer ${session.accessToken}`;
  }
  return config;
});
```

## 3) SSE（Server-Sent Events）の扱い

Professor の `/v1/qa/ask` はSSEでストリーミング応答を返します。

### Client Component での実装例
```typescript
// features/qa-chat/api/askQuestion.ts
export async function* streamAnswer(question: string, subjectId: string) {
  const response = await fetch('/api/qa/ask', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ question, subject_id: subjectId }),
  });
  
  const reader = response.body?.getReader();
  const decoder = new TextDecoder();
  
  while (true) {
    const { done, value } = await reader!.read();
    if (done) break;
    
    const chunk = decoder.decode(value);
    const events = parseSSE(chunk); // SSEパース処理
    
    for (const event of events) {
      yield event; // { type: 'thinking' | 'searching' | 'answer', data: ... }
    }
  }
}
```

### Route Handler でのプロキシ（推奨）
```typescript
// app/api/qa/ask/route.ts
export async function POST(request: Request) {
  const body = await request.json();
  
  // Professor APIへプロキシ（認証ヘッダー付与等）
  const professorResponse = await fetch(`${PROFESSOR_API_URL}/v1/qa/ask`, {
    method: 'POST',
    headers: { 
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${await getServerSession()}` // Phase 2
    },
    body: JSON.stringify(body),
  });
  
  // SSEをそのまま返す
  return new Response(professorResponse.body, {
    headers: { 'Content-Type': 'text/event-stream' },
  });
}
```

SSOT：`../03_integration/API_CONTRACT_WORKFLOW.md`

---

### エラーハンドリング統一

```typescript
// src/shared/api/error-handler.ts
import { ERROR_CODES } from './error-codes';

export function handleApiError(error: AxiosError) {
  const code = error.response?.data?.code;
  const message = ERROR_CODES[code] || 'エラーが発生しました';
  
  // トースト通知、エラーページ表示など
  showErrorToast(message);
}
```
