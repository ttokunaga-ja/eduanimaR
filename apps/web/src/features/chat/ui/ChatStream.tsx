'use client';

import { useCallback } from 'react';

import { useTranslation } from 'react-i18next';

import {
  postV1SubjectsSubjectIdChatsChatIdFeedback,
  PostV1SubjectsSubjectIdChatsChatIdFeedbackBodyRating,
} from '@/shared/api';

import type { ChatStreamState } from '../model/types';

interface ChatStreamProps {
  state: ChatStreamState;
}

/**
 * SSEストリーム表示コンポーネント。
 * thinking / searching / evidence / streaming / done / error 各フェーズを表示。
 */
export function ChatStream({ state }: ChatStreamProps) {
  const { t } = useTranslation('common');
  const { status, searchQuery, evidences, answer, chatId, error } = state;

  const handleFeedback = useCallback(
    async (rating: PostV1SubjectsSubjectIdChatsChatIdFeedbackBodyRating) => {
      if (!chatId) return;
      // subjectId は URL パスから取得（Phase 1 簡易実装）
      const pathParts = window.location.pathname.split('/');
      const subjectIdx = pathParts.indexOf('subjects');
      const subjectId =
        subjectIdx !== -1 ? pathParts[subjectIdx + 1] : 'unknown';
      try {
        await postV1SubjectsSubjectIdChatsChatIdFeedback(
          subjectId,
          chatId,
          { rating },
          { headers: { 'X-Dev-User': 'dev-user' } },
        );
      } catch {
        // フィードバック失敗はサイレントに無視
      }
    },
    [chatId],
  );

  if (status === 'idle') return null;

  return (
    <div style={styles.container}>
      {/* ─── ステータスインジケーター ─── */}
      {(status === 'thinking' || status === 'searching') && (
        <div style={styles.statusBanner}>
          <span style={styles.spinner} aria-hidden="true">{t('chat.spinner')}</span>
          {status === 'thinking' && t('chat.thinking')}
          {status === 'searching' && (
            <>
              {t('chat.searching')}
              {searchQuery && (
                <code style={styles.queryBadge}>
                  {t('chat.searchQueryOpen')}{searchQuery}{t('chat.searchQueryClose')}
                </code>
              )}
            </>
          )}
        </div>
      )}

      {/* ─── エビデンス（根拠資料） ─── */}
      {evidences.length > 0 && (
        <details style={styles.evidenceDetails} open>
          <summary style={styles.evidenceSummary}>
            {t('chat.evidenceTitle', { count: evidences.length })}
          </summary>
          <ul style={styles.evidenceList}>
            {evidences.map((ev, i) => (
              <li key={i} style={styles.evidenceItem}>
                <span style={styles.evidenceFile}>
                  {ev.file_name}
                  {ev.page_number > 0 && (
                    <span style={styles.evidencePage}>
                      {' '}{t('chat.page', { num: ev.page_number })}
                    </span>
                  )}
                </span>
                {ev.excerpt && (
                  <blockquote style={styles.evidenceExcerpt}>
                    {ev.excerpt}
                  </blockquote>
                )}
              </li>
            ))}
          </ul>
        </details>
      )}

      {/* ─── 回答テキスト（ストリーミング） ─── */}
      {(status === 'streaming' || status === 'done') && answer && (
        <div style={styles.answerBox}>
          <p style={styles.answerLabel}>{t('chat.answerLabel')}</p>
          <div style={styles.answerText}>
            {/* Markdown 未対応のためプレーンテキスト表示。Phase 2 で react-markdown 等を導入予定 */}
            {answer.split('\n').map((line, i) => (
              <span key={i}>
                {line}
                {i < answer.split('\n').length - 1 && <br />}
              </span>
            ))}
            {status === 'streaming' && (
              <span style={styles.cursor} aria-hidden="true">{t('chat.cursor')}</span>
            )}
          </div>
        </div>
      )}

      {/* ─── フィードバックボタン（done 時のみ） ─── */}
      {status === 'done' && chatId && (
        <div style={styles.feedbackRow}>
          <span style={styles.feedbackLabel}>{t('chat.feedbackPrompt')}</span>
          <button
            onClick={() => handleFeedback(PostV1SubjectsSubjectIdChatsChatIdFeedbackBodyRating.good)}
            style={styles.feedbackBtn}
            aria-label={t('chat.thumbsUp')}
            title={t('chat.thumbsUp')}
          >
            {t('chat.thumbsUp')}
          </button>
          <button
            onClick={() => handleFeedback(PostV1SubjectsSubjectIdChatsChatIdFeedbackBodyRating.bad)}
            style={styles.feedbackBtn}
            aria-label={t('chat.thumbsDown')}
            title={t('chat.thumbsDown')}
          >
            {t('chat.thumbsDown')}
          </button>
        </div>
      )}

      {/* ─── エラー表示 ─── */}
      {status === 'error' && (
        <div style={styles.errorBox} role="alert">
          <strong>{t('chat.errorLabel')}</strong>
          {' '}
          {error ?? t('chat.unknownError')}
        </div>
      )}
    </div>
  );
}

const styles = {
  container: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '12px',
    padding: '16px',
    flex: 1,
    overflowY: 'auto' as const,
  },
  statusBanner: {
    display: 'flex',
    alignItems: 'center',
    gap: '6px',
    padding: '8px 12px',
    background: '#e3f2fd',
    borderRadius: '6px',
    fontSize: '14px',
    color: '#1565c0',
  },
  spinner: {
    fontSize: '16px',
  },
  queryBadge: {
    display: 'inline-block',
    padding: '2px 6px',
    marginLeft: '4px',
    background: '#bbdefb',
    borderRadius: '4px',
    fontSize: '13px',
  },
  evidenceDetails: {
    border: '1px solid #e0e0e0',
    borderRadius: '6px',
    overflow: 'hidden',
  },
  evidenceSummary: {
    padding: '8px 12px',
    background: '#f5f5f5',
    cursor: 'pointer',
    fontSize: '13px',
    fontWeight: 600 as const,
    userSelect: 'none' as const,
  },
  evidenceList: {
    listStyle: 'none',
    margin: 0,
    padding: '8px 12px',
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '8px',
  },
  evidenceItem: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '4px',
  },
  evidenceFile: {
    fontSize: '13px',
    fontWeight: 600 as const,
    color: '#333',
  },
  evidencePage: {
    fontWeight: 400,
    color: '#666',
    fontSize: '12px',
  },
  evidenceExcerpt: {
    margin: '0',
    padding: '4px 8px',
    borderLeft: '3px solid #90caf9',
    background: '#f8fbff',
    fontSize: '12px',
    color: '#555',
    lineHeight: '1.5',
  },
  answerBox: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '6px',
  },
  answerLabel: {
    margin: 0,
    fontSize: '12px',
    fontWeight: 700 as const,
    color: '#666',
    textTransform: 'uppercase' as const,
    letterSpacing: '0.05em',
  },
  answerText: {
    fontSize: '15px',
    lineHeight: '1.7',
    color: '#212121',
    whiteSpace: 'pre-wrap' as const,
  },
  cursor: {
    display: 'inline-block',
    color: '#1976d2',
  },
  feedbackRow: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    paddingTop: '8px',
    borderTop: '1px solid #eee',
  },
  feedbackLabel: {
    fontSize: '13px',
    color: '#666',
  },
  feedbackBtn: {
    background: 'none',
    border: '1px solid #e0e0e0',
    borderRadius: '50%',
    width: '36px',
    height: '36px',
    fontSize: '18px',
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  },
  errorBox: {
    padding: '10px 14px',
    background: '#ffebee',
    border: '1px solid #ef9a9a',
    borderRadius: '6px',
    fontSize: '14px',
    color: '#c62828',
  },
} as const;
