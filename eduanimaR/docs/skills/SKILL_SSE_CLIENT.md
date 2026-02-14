# SKILL: SSE Client（Server-Sent Events / Streaming）

対象：SSEストリーミングを安全かつ効率的に実装する。

変化に敏感な領域：
- fetch API の ReadableStream 利用
- EventSource の制約（カスタムヘッダー不可）
- ブラウザ互換性

関連：
- `../03_integration/SSE_STREAMING.md`
- `../01_architecture/RESILIENCY.md`
- `../04_testing/TEST_STRATEGY.md`

---

## Versions（2026-02-15）

- ブラウザネイティブ API（fetch / ReadableStream）を使用
- ポリフィル不要（モダンブラウザはすべてサポート）

---

## Must

- fetch API の ReadableStream を使用する（EventSource は制約あり）
- 接続エラー・タイムアウトを適切にハンドリングする
- メモリリークを防ぐ（ストリーム終了時にクリーンアップ）
- UI 更新は React State で管理し、リアルタイム反映する

### 基本実装パターン

```typescript
// features/chat/lib/useSendQuestion.ts
import { useState, useCallback } from 'react';

export function useSendQuestion(threadId: string) {
  const [streamingMessage, setStreamingMessage] = useState<StreamingMessage | null>(null);
  const [isStreaming, setIsStreaming] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const sendQuestion = useCallback(async (question: string) => {
    setIsStreaming(true);
    setError(null);

    try {
      // 1. 質問送信
      const response = await fetch('/api/v1/questions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ thread_id: threadId, content: question }),
      });

      if (!response.ok) throw new Error('Failed to send question');

      const { request_id } = await response.json();

      // 2. SSE 接続
      const eventResponse = await fetch(`/api/v1/questions/${request_id}/events`);
      
      if (!eventResponse.ok) throw new Error('Failed to connect to SSE');
      if (!eventResponse.body) throw new Error('No response body');

      const reader = eventResponse.body.getReader();
      const decoder = new TextDecoder();

      let buffer = '';
      let contentBuffer = '';

      // 3. ストリーミング受信
      setStreamingMessage({
        message_id: null,
        role: 'assistant',
        content: '',
        isStreaming: true,
      });

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
              contentBuffer += data.content;
              setStreamingMessage((prev) => ({
                ...prev!,
                content: contentBuffer,
              }));
            } else if (data.type === 'done') {
              setStreamingMessage(null);
              setIsStreaming(false);
              // キャッシュに追加（TanStack Query）
              // ...
            } else if (data.type === 'error') {
              throw new Error(data.message);
            }
          }
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Unknown error'));
      setStreamingMessage(null);
      setIsStreaming(false);
    }
  }, [threadId]);

  return { sendQuestion, streamingMessage, isStreaming, error };
}
```

---

## エラーハンドリング

### 1) 接続エラー

- ネットワーク断、サーバーエラー等
- リトライ（最大3回、エクスポネンシャルバックオフ）

```typescript
const retryFetch = async (url: string, options: RequestInit, retries = 3) => {
  for (let i = 0; i < retries; i++) {
    try {
      return await fetch(url, options);
    } catch (err) {
      if (i === retries - 1) throw err;
      await new Promise((resolve) => setTimeout(resolve, 2 ** i * 1000));
    }
  }
};
```

### 2) タイムアウト

- 60秒以内にイベントが来ない場合はタイムアウト

```typescript
const timeout = setTimeout(() => {
  reader.cancel();
  throw new Error('SSE timeout');
}, 60000); // 60秒

// ストリーミング完了時にタイムアウトをクリア
clearTimeout(timeout);
```

### 3) パースエラー

- 不正な JSON が来た場合

```typescript
try {
  const data = JSON.parse(line.slice(6));
  // ...
} catch (err) {
  console.error('Failed to parse SSE data:', err);
  // エラーを記録するが、ストリーミングは継続
}
```

---

## メモリリーク対策

### 1) クリーンアップ

```typescript
useEffect(() => {
  return () => {
    // コンポーネントアンマウント時にストリームをキャンセル
    reader?.cancel();
  };
}, []);
```

### 2) AbortController の使用

```typescript
const abortController = new AbortController();

const eventResponse = await fetch(`/api/v1/questions/${request_id}/events`, {
  signal: abortController.signal,
});

// キャンセル時
abortController.abort();
```

---

## テストパターン

### 1) EventSource のモック

```typescript
// vitest.setup.ts
global.fetch = vi.fn();

// テスト内
vi.mocked(fetch).mockResolvedValueOnce({
  ok: true,
  body: new ReadableStream({
    start(controller) {
      controller.enqueue(new TextEncoder().encode('data: {"type":"content","content":"Hello"}\n\n'));
      controller.enqueue(new TextEncoder().encode('data: {"type":"done","message_id":"123"}\n\n'));
      controller.close();
    },
  }),
} as any);
```

---

## 禁止

- EventSource を使う（カスタムヘッダー不可のため）
- ストリーム終了時のクリーンアップを忘れる（メモリリーク）
- エラーハンドリングを省略する（ネットワーク断で固まる）
- UI 更新を setState 以外で行う（再レンダリング不整合）

---

## チェックリスト

- [ ] fetch + ReadableStream で SSE を実装しているか？
- [ ] 接続エラー・タイムアウトをハンドリングしているか？
- [ ] ストリーム終了時に `reader.cancel()` でクリーンアップしているか？
- [ ] UI 更新は React State で管理されているか？
- [ ] テストで ReadableStream のモックを実装しているか？
- [ ] エクスポネンシャルバックオフでリトライしているか？

---

## 参考

- MDN: ReadableStream - https://developer.mozilla.org/en-US/docs/Web/API/ReadableStream
- Server-Sent Events - https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events
