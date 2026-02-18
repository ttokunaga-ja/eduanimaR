'use client';

import { useState, useCallback, useRef } from 'react';

import type { ChatStreamState, SSEEvent } from './types';

const INITIAL_STATE: ChatStreamState = {
  status: 'idle',
  evidences: [],
  answer: '',
};

/**
 * SSE over POST でチャットストリームを管理するフック。
 *
 * POST /v1/subjects/{subjectId}/chats → text/event-stream
 * EventSource は POST をサポートしないため fetch + ReadableStream で実装。
 *
 * @param subjectId - 対象科目 ID
 */
export function useChatStream(subjectId: string) {
  const [state, setState] = useState<ChatStreamState>(INITIAL_STATE);
  const abortRef = useRef<AbortController | null>(null);

  const ask = useCallback(
    async (question: string) => {
      // 既存ストリームをキャンセル
      abortRef.current?.abort();
      const ac = new AbortController();
      abortRef.current = ac;

      setState({ status: 'thinking', evidences: [], answer: '' });

      const baseUrl = (
        process.env.NEXT_PUBLIC_API_BASE_URL ?? ''
      ).replace(/\/$/, '');
      const url = `${baseUrl}/v1/subjects/${subjectId}/chats`;

      try {
        const res = await fetch(url, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            // Phase 1: 固定devユーザー識別ヘッダー
            'X-Dev-User': 'dev-user',
          },
          body: JSON.stringify({ question }),
          signal: ac.signal,
          credentials: 'include',
        });

        if (!res.ok) {
          const text = await res.text().catch(() => '');
          setState((s) => ({
            ...s,
            status: 'error',
            error: `HTTP ${res.status}: ${text}`,
          }));
          return;
        }

        if (!res.body) {
          setState((s) => ({
            ...s,
            status: 'error',
            error: 'Response body is null',
          }));
          return;
        }

        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });
          // SSE行は "\n\n" で区切られるが、行単位で処理するため "\n" で分割
          const lines = buffer.split('\n');
          // 最後の不完全な行はバッファに残す
          buffer = lines.pop() ?? '';

          for (const line of lines) {
            if (!line.startsWith('data: ')) continue;
            const raw = line.slice(6).trim();
            if (!raw) continue;

            let event: SSEEvent;
            try {
              event = JSON.parse(raw) as SSEEvent;
            } catch {
              continue;
            }

            switch (event.type) {
              case 'thinking':
                setState((s) => ({ ...s, status: 'thinking' }));
                break;
              case 'searching':
                setState((s) => ({
                  ...s,
                  status: 'searching',
                  searchQuery: event.data.query,
                }));
                break;
              case 'evidence':
                setState((s) => ({
                  ...s,
                  evidences: event.data.chunks ?? [],
                }));
                break;
              case 'chunk':
                setState((s) => ({
                  ...s,
                  status: 'streaming',
                  answer: s.answer + event.data.text,
                }));
                break;
              case 'done':
                setState((s) => ({
                  ...s,
                  status: 'done',
                  chatId: event.data.chat_id,
                }));
                break;
              case 'error':
                setState((s) => ({
                  ...s,
                  status: 'error',
                  error: event.data.message,
                }));
                break;
            }
          }
        }
      } catch (err) {
        if (err instanceof Error && err.name === 'AbortError') return;
        setState((s) => ({ ...s, status: 'error', error: String(err) }));
      }
    },
    [subjectId],
  );

  const reset = useCallback(() => {
    abortRef.current?.abort();
    setState(INITIAL_STATE);
  }, []);

  return { state, ask, reset };
}
