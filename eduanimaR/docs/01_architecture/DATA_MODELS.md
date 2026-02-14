# Data Models & Type Definitions（Contract）

このドキュメントは、フロントエンドで扱うデータモデルと型定義の契約を固定し、OpenAPI 由来の型を SSOT（Single Source of Truth）として運用します。

**重要**: このドキュメントは Backend DB Schema (`../../eduanimaR_Professor/docs/01_architecture/DB_SCHEMA_TABLES.md`) と整合性を保つ必要があります。

関連：
- Backend DB Schema：`../../eduanimaR_Professor/docs/01_architecture/DB_SCHEMA_TABLES.md`（SSOT）
- API生成：`../03_integration/API_GEN.md`
- Orval設定：`../skills/SKILL_ORVAL_OPENAPI.md`
- キャッシュ戦略：`CACHING_STRATEGY.md`
- TanStack Query：`../02_tech_stack/STATE_QUERY.md`
- 認証：`../03_integration/AUTH_SESSION.md`

---

## 結論（Must）

- 主要エンティティ: **User**, **Subject**, **RawFile**, **Material**, **Chat**
- OpenAPI 由来の型を **SSOT**（Single Source of Truth）とする
- **Backend DB Schema と完全一致**させる（nanoid, ENUM型等）
- UI 専用の補助型は最小限に留める
- TanStack Query のキャッシュキー規約を統一する
- 楽観的更新（Optimistic Update）のパターンを標準化する

---

## 1) 主要エンティティ

### 1.1 User（ユーザー）

```typescript
// src/shared/api/generated/types.ts (OpenAPI由来)
export interface User {
  id: string;                // UUID (内部ID)
  nanoid: string;            // 20文字の外部公開ID
  provider: string;          // OAuth/OIDCプロバイダ（例: "google", "microsoft"）
  provider_user_id: string;  // プロバイダ側のユーザーID
  role: UserRole;            // "student" | "instructor" | "admin"
  created_at: string;        // ISO 8601
  updated_at: string;        // ISO 8601
  last_login_at?: string;    // ISO 8601
  is_active: boolean;        // ソフトデリート用
}

export type UserRole = 'student' | 'instructor' | 'admin';
```

説明：
- **個人情報非収集**: `email`, `display_name` は含まれない（Backend DBにも存在しない）
- `nanoid` は外部公開ID（URL/ログ/問い合わせ用、20文字固定）
- `provider` / `provider_user_id` で複数IdP対応
- フロントエンドでユーザー表示が必要な場合は `provider` + `provider_user_id` の組み合わせか、別途UIで入力させる

### 1.2 Subject（科目）

```typescript
// src/shared/api/generated/types.ts (OpenAPI由来)
export interface Subject {
  id: string;                // UUID (内部ID)
  nanoid: string;            // 20文字の外部公開ID
  owner_user_id: string;     // 所有者のUser UUID
  title: string;             // 科目タイトル（例: "データベース基礎"）
  description?: string;      // 科目説明
  academic_year?: string;    // 例: "2026"
  semester?: string;         // 例: "Spring"
  course_code?: string;      // 例: "CS101"
  created_at: string;        // ISO 8601
  updated_at: string;        // ISO 8601
  is_active: boolean;        // ソフトデリート用
}
```

説明：
- ユーザーが管理する科目（授業、コース）
- **1科目 = 1つのスコープ境界**（物理制約の基準）
- 資料（RawFile）とチャット（Chat）の親エンティティ
- 削除時は関連する RawFile / Chat も削除される（カスケード）

### 1.3 RawFile（原本ファイル）

```typescript
// src/shared/api/generated/types.ts (OpenAPI由来)
export interface RawFile {
  id: string;                // UUID (内部ID)
  nanoid: string;            // 20文字の外部公開ID
  user_id: string;           // 所有者のUser UUID（物理制約）
  subject_id: string;        // 所属科目のSubject UUID（物理制約）
  original_filename: string; // 元のファイル名（例: "lecture01.pdf"）
  file_type: FileType;       // ファイルタイプ（ENUM）
  file_size_bytes: number;   // ファイルサイズ（バイト）
  source_url?: string;       // 自動取り込み用ソースURL（将来機能）
  status: FileStatus;        // ファイルステータス
  total_pages?: number;      // PDF/PowerPointのページ数
  mime_type?: string;        // MIMEタイプ
  processed_at?: string;     // 処理完了日時（ISO 8601）
  created_at: string;        // ISO 8601
  updated_at: string;        // ISO 8601
  is_active: boolean;        // ソフトデリート用
}

// Gemini APIがサポートする形式（Backend DB ENUM型と一致）
export type FileType =
  // ドキュメント（ネイティブ対応）
  | 'pdf'
  | 'text'
  // スクリプト・コード（ネイティブ対応）
  | 'python'
  | 'go'
  | 'javascript'
  | 'html'
  | 'css'
  | 'json'
  | 'markdown'
  | 'csv'
  // 画像（ネイティブ対応）
  | 'png'
  | 'jpeg'
  | 'webp'
  | 'heic'
  | 'heif'
  // MS Office（Drive API経由で変換が必要）
  | 'docx'
  | 'xlsx'
  | 'pptx'
  // Google Workspace（Drive API経由で変換が必要）
  | 'google_docs'
  | 'google_sheets'
  | 'google_slides'
  // その他
  | 'other';

export type FileStatus =
  | 'uploading'      // アップロード中
  | 'uploaded'       // アップロード完了
  | 'processing'     // 処理中（Vision Reasoning実行中）
  | 'completed'      // 処理完了
  | 'failed'         // 処理失敗
  | 'archived';      // アーカイブ済み
```

説明：
- 科目に紐づく原本ファイル
- **user_id + subject_id は必須**（物理制約の境界）
- アップロード後、Professor 側で解析処理が非同期実行される
- `status` が `completed` になるまで、検索対象にならない
- GCS に保存される（フロントエンドは署名付きURL経由でアクセス）

### 1.4 Material（資料チャンク）

```typescript
// src/shared/api/generated/types.ts (OpenAPI由来)
export interface Material {
  id: string;                // UUID (内部ID、NanoID不要)
  raw_file_id: string;       // 親ファイルのRawFile UUID
  sequence_in_file: number;  // ファイル内の順序
  page_start?: number;       // 元ページ範囲（開始）
  page_end?: number;         // 元ページ範囲（終了）
  content_markdown: string;  // チャンク内容（Markdown形式）
  char_count: number;        // 文字数
  embedding_model: string;   // Embeddingモデル（デフォルト: "text-embedding-004"）
  created_at: string;        // ISO 8601
  updated_at: string;        // ISO 8601
  is_active: boolean;        // ソフトデリート用
}
```

説明：
- 原本ファイルから生成された**意味単位のチャンク**
- Gemini 3 Flashで分割、Markdown化済み
- ベクトル埋め込み済み（検索に使用、フロントエンドには不要）
- `sequence_in_file` でファイル内の順序を保持（前後文脈の取得に使用）
- **NanoID不要**（内部処理のみ、外部参照は通常raw_fileレベル）

### 1.5 Chat（質問・検索セッション）

```typescript
// src/shared/api/generated/types.ts (OpenAPI由来)
export interface Chat {
  id: string;                // UUID (内部ID)
  nanoid: string;            // 20文字の外部公開ID
  user_id: string;           // 質問者のUser UUID（物理制約）
  subject_id: string;        // 所属科目のSubject UUID（物理制約）
  question: string;          // 質問内容
  plan_json?: PlanJson;      // Phase 2（Plan）結果（Structured Outputs JSON）
  termination_reason?: string; // 検索終了理由（実際の終了理由を記録）
  final_answer_markdown?: string; // Phase 4（Answer）結果
  feedback?: ChatFeedback;   // ユーザーフィードバック
  feedback_at?: string;      // フィードバック日時（ISO 8601）
  actual_search_steps: number; // 実際の検索ステップ数
  evidence_snippets?: EvidenceSnippet[]; // 根拠スニペット（JSONB配列）
  used_raw_file_ids: string[]; // 最終回答に使用したファイルUUID配列
  created_at: string;        // ISO 8601
  updated_at: string;        // ISO 8601
  completed_at?: string;     // 完了日時（ISO 8601）
  is_active: boolean;        // ソフトデリート用
}

export type ChatFeedback = 'good' | 'bad';

// Phase 2の計画全体
export interface PlanJson {
  investigation_items: InvestigationItem[];
  termination_conditions: TerminationConditions;
  search_strategy: SearchStrategy;
}

export interface InvestigationItem {
  task_id: string;
  description: string;
}

export interface TerminationConditions {
  max_search_steps: number;
  min_evidence_count: number;
  confidence_threshold: number;
  stop_reasons: string[];
}

export interface SearchStrategy {
  mode: SearchMode;  // "keyword" | "vector" | "hybrid"
  // その他の戦略パラメータ
}

export type SearchMode = 'keyword' | 'vector' | 'hybrid';

// 根拠スニペット（Backend JSONB構造と一致）
export interface EvidenceSnippet {
  material_id: string;        // materials.id (UUID)
  snippet: string;            // 抽出されたテキストスニペット
  task_id: string;            // Phase 2で定義された調査タスクID
  relevance_score: number;    // 関連度スコア（0.0〜1.0）
  page_start?: number;        // 元ドキュメントのページ範囲（開始）
  page_end?: number;          // 元ドキュメントのページ範囲（終了）
  search_step: number;        // どの検索ステップで取得されたか（1〜5）
  selected_for_answer: boolean; // 最終回答生成に使用されたか
}
```

説明：
- Q&A の質問・検索セッション（Phase 2〜4の統合テーブル）
- **Thread/Message モデルは廃止**（Backend には存在しない）
- `plan_json` にPhase 2の計画（調査項目、終了条件、検索戦略）を保存
- `evidence_snippets` でチャンク単位の詳細根拠を保存（JSONB配列）
- `used_raw_file_ids` でファイル単位の引用リストを保存（UUID配列）
- 1つのChatに対して1つの質問と1つの回答

---

## 2) OpenAPI を SSOT とする運用

### 2.1 方針

- Professor の OpenAPI 仕様書（`../../eduanimaR_Professor/docs/openapi.yaml`）を正とする
- **Backend DB Schema** (`DB_SCHEMA_TABLES.md`) と OpenAPI が一致していることを確認
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
// src/entities/user/index.ts
export type { User, UserRole } from '@/shared/api/generated/types';

// src/entities/subject/index.ts
export type { Subject } from '@/shared/api/generated/types';

// src/entities/rawFile/index.ts
export type { RawFile, FileType, FileStatus } from '@/shared/api/generated/types';

// src/entities/chat/index.ts
export type { Chat, ChatFeedback, EvidenceSnippet, PlanJson } from '@/shared/api/generated/types';
```

---

## 3) UI 専用の補助型（最小限）

OpenAPI 由来の型で不足する場合のみ、UI 専用の補助型を定義する。

### 3.1 例：StreamingChat

SSE ストリーミング中の部分的なチャット回答を扱う型：

```typescript
// src/features/chat/types.ts
export interface StreamingChat {
  chat_id: string | null;     // 確定前は null
  question: string;           // 質問内容
  answer_content: string;     // 逐次追加される回答
  evidence_snippets?: EvidenceSnippet[]; // 確定後に追加される
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
['users', userId]                      // 特定ユーザー
['subjects']                           // 全科目
['subjects', subjectId]                // 特定科目
['rawFiles', { subjectId }]            // 科目別ファイル一覧
['rawFiles', rawFileId]                // 特定ファイル
['chats', { subjectId }]               // 科目別チャット一覧
['chats', chatId]                      // 特定チャット
```

### 4.2 キャッシュキーの定義場所

```typescript
// src/shared/api/queryKeys.ts
export const queryKeys = {
  users: {
    detail: (id: string) => ['users', id] as const,
  },
  subjects: {
    all: ['subjects'] as const,
    detail: (id: string) => ['subjects', id] as const,
  },
  rawFiles: {
    bySubject: (subjectId: string) => ['rawFiles', { subjectId }] as const,
    detail: (id: string) => ['rawFiles', id] as const,
  },
  chats: {
    bySubject: (subjectId: string) => ['chats', { subjectId }] as const,
    detail: (id: string) => ['chats', id] as const,
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
        old.filter((s) => s.id !== subjectId)
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
   - `POST /v1/chats` → `202 Accepted`
   - `chat_id` と `events_url` を取得

2. **SSE 接続**
   - `GET /v1/chats/{chat_id}/events`
   - イベントを逐次受信：`data: { type: "content", content: "..." }`

3. **ストリーミング状態の管理**
   - React State で `StreamingChat` を更新
   - 確定後に TanStack Query のキャッシュに追加

### 6.2 実装例

```typescript
// src/features/chat/lib/useSendQuestion.ts
export function useSendQuestion(subjectId: string) {
  const [streamingChat, setStreamingChat] = useState<StreamingChat | null>(null);
  const queryClient = useQueryClient();

  const sendQuestion = async (question: string) => {
    // 1. 質問送信
    const { chat_id, events_url } = await postChat({ 
      subject_id: subjectId, 
      question 
    });

    // 2. SSE 接続（fetch + ReadableStream）
    const response = await fetch(events_url);
    if (!response.ok || !response.body) {
      throw new Error('Failed to connect to SSE');
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();

    // 3. ストリーミング状態を初期化
    setStreamingChat({
      chat_id,
      question,
      answer_content: '',
      isStreaming: true,
    });

    let buffer = '';

    // 4. イベント受信
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop() || '';

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const data = JSON.parse(line.slice(6));

          if (data.type === 'content') {
            setStreamingChat((prev) => ({
              ...prev!,
              answer_content: prev!.answer_content + data.content,
            }));
          } else if (data.type === 'done') {
            // 5. 確定後、キャッシュに追加
            const finalChat: Chat = {
              id: chat_id,
              nanoid: data.nanoid,
              user_id: data.user_id,
              subject_id: subjectId,
              question,
              final_answer_markdown: data.answer,
              evidence_snippets: data.evidence_snippets,
              used_raw_file_ids: data.used_raw_file_ids,
              actual_search_steps: data.actual_search_steps,
              created_at: new Date().toISOString(),
              updated_at: new Date().toISOString(),
              completed_at: new Date().toISOString(),
              is_active: true,
            };

            queryClient.setQueryData(
              queryKeys.chats.bySubject(subjectId),
              (old: Chat[] = []) => [...old, finalChat]
            );

            setStreamingChat(null);
          }
        }
      }
    }
  };

  return { sendQuestion, streamingChat };
}
```

---

## 7) NanoID の扱い

### 7.1 NanoID 20文字の適用基準

Backend DB Schemaに従い、以下のエンティティは **20文字のNanoID** を持つ：

- `users`
- `subjects`
- `raw_files`
- `chats`

**NanoID不要**：
- `materials`（内部処理のみ、外部参照は通常raw_fileレベル）
- `jobs`（内部処理のみ）

### 7.2 フロントエンドでの使用

- URLパラメータには NanoID を使用（例: `/files/:nanoid`）
- 内部処理（API呼び出し、キャッシュキー）には UUID を使用
- 表示時は NanoID を優先（ユーザーフレンドリー）

```typescript
// 例：ファイル詳細ページ
// URL: /files/ABC123XYZ456QWERTY78 (nanoid)
// API: GET /v1/raw_files/{uuid}
// キャッシュキー: ['rawFiles', uuid]
```

---

## 禁止（AI/人間共通）

- OpenAPI 生成コード（`src/shared/api/generated/`）を手編集する
- Backend DB Schema と異なる型定義を作成する（必ずDB_SCHEMA_TABLES.mdを参照）
- UI 専用の補助型を無秩序に増やす（必要最小限に留める）
- キャッシュキーを統一せず、場当たり的に定義する
- 楽観的更新でロールバック処理を省略する（エラー時に不整合が起きる）
- SSE ストリーミング中のメッセージをキャッシュに直接追加する（確定後に追加）
- **Thread/Message モデルを使用する**（Backend には存在しない、Chat モデルを使用）
- **display_name を User モデルに含める**（Backend DBに存在しない）

---

## 実装チェックリスト

- [ ] Backend DB Schema (`DB_SCHEMA_TABLES.md`) を最新版で確認したか？
- [ ] OpenAPI 仕様書が Backend DB Schema と一致しているか？
- [ ] `npm run api:generate` で型を生成したか？
- [ ] 生成コードを手編集していないか？
- [ ] NanoID (20文字) が必要なエンティティに含まれているか？
- [ ] file_type ENUM が Backend の完全なリストと一致しているか？
- [ ] Chat モデルを使用し、Thread/Message モデルを使用していないか？
- [ ] evidence_snippets が Backend JSONB構造と一致しているか？
- [ ] キャッシュキーが `queryKeys` で統一管理されているか？
- [ ] 楽観的更新でロールバック処理が実装されているか？
- [ ] SSE ストリーミングのデータフローが正しく実装されているか？
- [ ] UI 専用の補助型が必要最小限か？
- [ ] FSD の Public API で型が再エクスポートされているか？
