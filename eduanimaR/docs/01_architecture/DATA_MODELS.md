# Data Models & Type Definitions（Contract）

このドキュメントは、フロントエンドで扱うデータモデルと型定義の契約を固定し、OpenAPI 由来の型を SSOT（Single Source of Truth）として運用します。

関連：
- API生成：`../03_integration/API_GEN.md`
- Orval設定：`../skills/SKILL_ORVAL_OPENAPI.md`
- キャッシュ戦略：`CACHING_STRATEGY.md`
- TanStack Query：`../02_tech_stack/STATE_QUERY.md`
- 認証：`../03_integration/AUTH_SESSION.md`

---

## 結論（Must）

- 主要エンティティ: **Subject**, **File**, **Thread**, **Message**, **Source**
- OpenAPI 由来の型を **SSOT**（Single Source of Truth）とする
- UI 専用の補助型は最小限に留める
- TanStack Query のキャッシュキー規約を統一する
- 楽観的更新（Optimistic Update）のパターンを標準化する

---

## 1) 主要エンティティ

### 1.1 Subject（科目）

```typescript
// src/shared/api/generated/types.ts (OpenAPI由来)
export interface Subject {
  subject_id: string;        // UUID
  display_name: string;      // "データベース基礎"
  created_at: string;        // ISO 8601
  updated_at: string;        // ISO 8601
}
```

説明：
- ユーザーが管理する科目（授業、コース）
- 資料（File）とスレッド（Thread）の親エンティティ
- 削除時は関連する File / Thread も削除される（カスケード）

### 1.2 File（資料）

```typescript
// src/shared/api/generated/types.ts (OpenAPI由来)
export interface File {
  file_id: string;           // UUID
  subject_id: string;        // 所属科目
  filename: string;          // "lecture01.pdf"
  file_type: FileType;       // "pdf" | "pptx" | "docx" | "txt"
  file_size: number;         // バイト数
  upload_status: UploadStatus; // "pending" | "processing" | "completed" | "failed"
  uploaded_at: string;       // ISO 8601
}

export type FileType = 'pdf' | 'pptx' | 'docx' | 'txt';
export type UploadStatus = 'pending' | 'processing' | 'completed' | 'failed';
```

説明：
- 科目に紐づく資料ファイル
- アップロード後、Professor 側で解析処理が非同期実行される
- `upload_status` が `completed` になるまで、検索対象にならない

### 1.3 Thread（Q&Aスレッド）

```typescript
// src/shared/api/generated/types.ts (OpenAPI由来)
export interface Thread {
  thread_id: string;         // UUID
  subject_id: string;        // 所属科目
  title: string;             // 質問の要約（最初のメッセージから生成）
  created_at: string;        // ISO 8601
  updated_at: string;        // ISO 8601
}
```

説明：
- Q&A のスレッド（会話の単位）
- 1つのスレッドに複数の Message が紐づく
- タイトルは最初の質問から自動生成される

### 1.4 Message（メッセージ）

```typescript
// src/shared/api/generated/types.ts (OpenAPI由来)
export interface Message {
  message_id: string;        // UUID
  thread_id: string;         // 所属スレッド
  role: MessageRole;         // "user" | "assistant"
  content: string;           // メッセージ本文
  sources?: Source[];        // AI回答の根拠（assistant のみ）
  created_at: string;        // ISO 8601
}

export type MessageRole = 'user' | 'assistant';
```

説明：
- スレッド内の個別メッセージ
- `role` が `user` の場合：ユーザーの質問
- `role` が `assistant` の場合：AI の回答（Source 付き）

### 1.5 Source（根拠）

```typescript
// src/shared/api/generated/types.ts (OpenAPI由来)
export interface Source {
  source_id: string;         // UUID
  file_id: string;           // 参照元ファイル
  filename: string;          // "lecture01.pdf"
  page_number?: number;      // ページ番号（PDFの場合）
  excerpt: string;           // 引用箇所（前後100文字程度）
  relevance_score: number;   // 関連度スコア（0.0〜1.0）
}
```

説明：
- AI 回答の根拠となる資料の引用箇所
- ユーザーがクリックすると、該当ファイルの該当箇所を表示できる
- `relevance_score` でソート表示する

---

## 2) OpenAPI を SSOT とする運用

### 2.1 方針

- Professor の OpenAPI 仕様書（`../../eduanimaR_Professor/docs/openapi.yaml`）を正とする
- Orval でフロントエンドの型を自動生成する（`npm run api:generate`）
- **生成コードは手編集禁止**：変更が必要なら OpenAPI 側を修正する

### 2.2 生成先

```
src/shared/api/generated/
  types.ts          # 型定義
  api.ts            # API 関数
  hooks.ts          # TanStack Query Hooks
```

### 2.3 型の再エクスポート

FSD の Public API として再エクスポートする：

```typescript
// src/entities/subject/index.ts
export type { Subject } from '@/shared/api/generated/types';
```

---

## 3) UI 専用の補助型（最小限）

OpenAPI 由来の型で不足する場合のみ、UI 専用の補助型を定義する。

### 3.1 例：StreamingMessage

SSE ストリーミング中の部分的なメッセージを扱う型：

```typescript
// src/features/chat/types.ts
export interface StreamingMessage {
  message_id: string | null;  // 確定前は null
  role: 'assistant';
  content: string;            // 逐次追加される
  sources?: Source[];         // 確定後に追加される
  isStreaming: boolean;       // ストリーミング中フラグ
}
```

### 3.2 例：FileWithProgress

アップロード中のファイルを扱う型：

```typescript
// src/features/fileUpload/types.ts
export interface FileWithProgress {
  file: File;                 // ブラウザの File オブジェクト
  progress: number;           // 0〜100
  status: 'uploading' | 'completed' | 'failed';
  error?: string;             // エラーメッセージ
}
```

---

## 4) TanStack Query のキャッシュキー規約

### 4.1 基本規約

キャッシュキーは以下の形式で統一する：

```typescript
[entity, ...identifiers, ...filters]
```

例：
```typescript
['subjects']                           // 全科目
['subjects', subjectId]                // 特定科目
['threads', { subjectId }]             // 科目別スレッド一覧
['threads', threadId]                  // 特定スレッド
['messages', { threadId }]             // スレッド別メッセージ一覧
['files', { subjectId }]               // 科目別ファイル一覧
['files', fileId]                      // 特定ファイル
```

### 4.2 キャッシュキーの定義場所

```typescript
// src/shared/api/queryKeys.ts
export const queryKeys = {
  subjects: {
    all: ['subjects'] as const,
    detail: (id: string) => ['subjects', id] as const,
  },
  threads: {
    bySubject: (subjectId: string) => ['threads', { subjectId }] as const,
    detail: (id: string) => ['threads', id] as const,
  },
  messages: {
    byThread: (threadId: string) => ['messages', { threadId }] as const,
  },
  files: {
    bySubject: (subjectId: string) => ['files', { subjectId }] as const,
    detail: (id: string) => ['files', id] as const,
  },
} as const;
```

---

## 5) 楽観的更新（Optimistic Update）のパターン

### 5.1 原則

- 重要な操作（削除、更新）は楽観的更新を実装し、UX を向上させる
- ネットワークエラー時はロールバックする
- 成功時はサーバーの最新データで置き換える

### 5.2 実装例：科目の削除

```typescript
// src/features/subjectManagement/lib/useDeleteSubject.ts
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '@/shared/api/queryKeys';
import { deleteSubject } from '@/shared/api/generated/api';

export function useDeleteSubject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (subjectId: string) => deleteSubject(subjectId),

    // 楽観的更新
    onMutate: async (subjectId) => {
      // 進行中のクエリをキャンセル
      await queryClient.cancelQueries({ queryKey: queryKeys.subjects.all });

      // 現在のキャッシュを保存（ロールバック用）
      const previousSubjects = queryClient.getQueryData(queryKeys.subjects.all);

      // キャッシュから削除
      queryClient.setQueryData(queryKeys.subjects.all, (old: Subject[]) =>
        old.filter((s) => s.subject_id !== subjectId)
      );

      return { previousSubjects };
    },

    // エラー時：ロールバック
    onError: (err, subjectId, context) => {
      queryClient.setQueryData(queryKeys.subjects.all, context?.previousSubjects);
    },

    // 成功時：再検証
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.subjects.all });
    },
  });
}
```

---

## 6) SSE ストリーミングのデータフロー

### 6.1 フロー

1. **質問送信**
   - `POST /v1/questions` → `202 Accepted`
   - `request_id` を取得

2. **SSE 接続**
   - `GET /v1/questions/{request_id}/events`
   - イベントを逐次受信：`data: { type: "content", content: "..." }`

3. **ストリーミング状態の管理**
   - React State で `StreamingMessage` を更新
   - 確定後に TanStack Query のキャッシュに追加

### 6.2 実装例

```typescript
// src/features/chat/lib/useSendQuestion.ts
export function useSendQuestion(threadId: string) {
  const [streamingMessage, setStreamingMessage] = useState<StreamingMessage | null>(null);
  const queryClient = useQueryClient();

  const sendQuestion = async (question: string) => {
    // 1. 質問送信
    const { request_id } = await postQuestion({ thread_id: threadId, content: question });

    // 2. SSE 接続
    const eventSource = new EventSource(`/api/v1/questions/${request_id}/events`);

    // 3. ストリーミング状態を初期化
    setStreamingMessage({
      message_id: null,
      role: 'assistant',
      content: '',
      isStreaming: true,
    });

    // 4. イベント受信
    eventSource.onmessage = (event) => {
      const data = JSON.parse(event.data);

      if (data.type === 'content') {
        setStreamingMessage((prev) => ({
          ...prev!,
          content: prev!.content + data.content,
        }));
      } else if (data.type === 'done') {
        // 5. 確定後、キャッシュに追加
        const finalMessage: Message = {
          message_id: data.message_id,
          thread_id: threadId,
          role: 'assistant',
          content: data.content,
          sources: data.sources,
          created_at: new Date().toISOString(),
        };

        queryClient.setQueryData(
          queryKeys.messages.byThread(threadId),
          (old: Message[]) => [...old, finalMessage]
        );

        setStreamingMessage(null);
        eventSource.close();
      }
    };
  };

  return { sendQuestion, streamingMessage };
}
```

---

## 禁止（AI/人間共通）

- OpenAPI 生成コード（`src/shared/api/generated/`）を手編集する
- UI 専用の補助型を無秩序に増やす（必要最小限に留める）
- キャッシュキーを統一せず、場当たり的に定義する
- 楽観的更新でロールバック処理を省略する（エラー時に不整合が起きる）
- SSE ストリーミング中のメッセージをキャッシュに直接追加する（確定後に追加）

---

## 実装チェックリスト

- [ ] OpenAPI 仕様書が最新か？
- [ ] `npm run api:generate` で型を生成したか？
- [ ] 生成コードを手編集していないか？
- [ ] キャッシュキーが `queryKeys` で統一管理されているか？
- [ ] 楽観的更新でロールバック処理が実装されているか？
- [ ] SSE ストリーミングのデータフローが正しく実装されているか？
- [ ] UI 専用の補助型が必要最小限か？
- [ ] FSD の Public API で型が再エクスポートされているか？
