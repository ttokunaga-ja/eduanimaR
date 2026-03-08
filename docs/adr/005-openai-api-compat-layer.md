# ADR 005: OpenAI API 互換レイヤー（professor サービス）

## Status
Accepted

## Date
2026-03-08

## Context

eduanimaR の professor サービスは独自のドメイン特化 REST API（`/api/v1/subjects/:id/chats` 等）を提供しているが、
評価ツール（pageBench）や外部クライアントが OpenAI Python/TypeScript SDK を使って接続できると、
クライアント側のコードを大幅に削減できる。

一方で以下の差異が存在する：

| 差異 | professor (現行) | OpenAI 標準 |
|------|----------------|------------|
| **コレクション概念** | `subject_id` (URL パス) | なし |
| **リクエスト形式** | `{"question": "..."}` | `{"messages": [...], "model": "..."}` |
| **SSE イベント形式** | `{"type":"answer","data":{"text":"..."}}` | `{"choices":[{"delta":{"content":"..."}}]}` |
| **中間イベント** | thinking / searching / evidence | なし |
| **ファイルアップロード** | `POST /subjects/:id/materials` | `POST /files` |
| **ファイルステータス** | `pending/processing/ready/failed` | `uploaded/processed/error` |
| **認証** | Phase 1: DevUser / Phase 2: Bearer JWT | Bearer API Key |

## Decision

### 1. スコープ付きパス方式（Approach C）を採用

新たに以下のエンドポイントを professor に追加する：

```
POST /api/v1/subjects/:subject_id/chat/completions  ← OpenAI Chat API 互換
POST /api/v1/subjects/:subject_id/files             ← OpenAI Files API 互換
GET  /api/v1/subjects/:subject_id/files/:file_id    ← OpenAI File ステータス互換
```

クライアントは `base_url` を以下に設定するだけで OpenAI SDK がそのまま動作する：

```python
client = OpenAI(
    base_url="https://professor.eduanima.ai/api/v1/subjects/{subject_id}",
    api_key="<Firebase ID Token or dev-token>"
)
```

**採用理由：**
- `subject_id` が URL に明示されるため、サーバーサイドでの parse が不要
- OpenAI SDK は `{base_url}/chat/completions`・`{base_url}/files` を自動的に呼ぶため追加設定不要
- 既存の `subjects/:id/*` ルートと同居でき、ルーティングが一貫する
- セキュリティ上、subject の所有権チェックは既存ユースケース層で担保済み

**不採用案：**
- `model` フィールド埋め込み（`"model": "professor/{subject_id}"`）: 意味的ミスマッチ
- `X-Subject-ID` カスタムヘッダー：SDK の `default_headers` 設定が必要で純粋な互換でない
- クエリパラメータ：OpenAI SDK が自動付与しない

### 2. 認証の互換性

```
Phase 1 (development):  DevUser ミドルウェア → Bearer の内容は任意（無視）
Phase 2 (production):   RequireJWT → Bearer <Firebase ID Token> ← OpenAI SDK 互換
```

`Authorization: Bearer <token>` は OpenAI SDK が標準で送るヘッダーと完全に一致する。
Phase 2 では既存の `RequireJWT` ミドルウェアを変更なしで使用できる。

### 3. SSE フォーマット変換

#### streaming (stream: true)

eduanimaR の各 SSE イベントを以下の OpenAI delta chunk に変換する：

| eduanimaR イベント | OpenAI 変換 |
|------------------|------------|
| `thinking` | `choices[0].delta.content = null` + `"eduanima_event": {"type":"thinking","data":{...}}` |
| `searching` | `choices[0].delta.content = null` + `"eduanima_event": {"type":"searching","data":{...}}` |
| `evidence` | `choices[0].delta.content = null` + `"eduanima_event": {"type":"evidence","data":{...}}` |
| `answer` | `choices[0].delta.content = "<text>"` (標準 OpenAI delta) |
| `done` | `choices[0].delta = {}`, `finish_reason = "stop"` → `data: [DONE]` |
| `error` | `{"error":{"message":"..."}}` + ストリーム終了 |

`eduanima_event` は拡張フィールドであり、標準の OpenAI SDK はこれを無視する。
eduanimaR 対応クライアント（pageBench 等）は `eduanima_event` を読んで中間状態を表示できる。

#### non-streaming (stream: false または省略)

全 `answer` チャンクをバッファして一つの `chat.completion` オブジェクトとして返す：

```json
{
  "id": "chatcmpl-{sessionID}",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "<requested model>",
  "choices": [{
    "index": 0,
    "message": {"role": "assistant", "content": "<full answer>"},
    "finish_reason": "stop"
  }],
  "usage": {"prompt_tokens": 0, "completion_tokens": 0, "total_tokens": 0},
  "eduanima_sources": [{"file_name": "...", "excerpt": "...", "why_relevant": "..."}]
}
```

`eduanima_sources` は非ストリーミング時のみ付与される拡張フィールド。

### 4. ファイル API の変換

#### ID フォーマット

- professor 内部: UUID with hyphens (`550e8400-e29b-41d4-a716-446655440000`)
- OpenAI 互換 ID: `file-` プレフィックス + ハイフンなし UUID (`file-550e8400e29b41d4a716446655440000`)

GET /:file_id では両フォーマットを受け付ける。

#### ステータスマッピング

| domain.FileStatus | OpenAI status |
|------------------|---------------|
| `pending` | `"uploaded"` |
| `processing` | `"uploaded"` |
| `ready` | `"processed"` |
| `failed` | `"error"` |

## Consequences

### Positive
- pageBench の `AGENT_BASE_URL` を変更するだけで eduanimaR に接続可能
- 外部の OpenAI SDK ベースクライアントが無改造で使用可能
- 既存の professor API は変更不要（完全後方互換）
- 認証方式が Phase 2 でも変更不要

### Negative
- `model` フィールドはサーバー側で無視される（意図を明示するコメントを残す）
- `usage` フィールドのトークンカウントは 0 になる（professor は Gemini を使用するため OpenAI トークンカウントは不明）
- `/v1/models` エンドポイントは未実装（SDK 初期化時に警告が出る場合があるが動作に影響しない）

## Related

- ADR 001: Contract Canonical Paths
- ADR 004: Librarian Proto Shim
- contracts/openapi/professor.yaml（本 ADR に基づいて更新）
