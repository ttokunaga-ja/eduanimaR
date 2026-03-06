'use client';

import { useTranslation } from 'react-i18next';

import { ChatInput, ChatStream, useChatStream } from '@/features/chat';

interface QAPageProps {
  subjectId: string;
}

/**
 * QA（質問応答）ページ。
 * features/chat の ChatInput / ChatStream / useChatStream を組み合わせる。
 *
 * レイアウト:
 *   ┌─────────────────────┐
 *   │ ヘッダー            │
 *   ├─────────────────────┤
 *   │ ストリーム表示      │  ← flex-grow
 *   ├─────────────────────┤
 *   │ 入力フォーム        │
 *   └─────────────────────┘
 */
export function QAPage({ subjectId }: QAPageProps) {
  const { t } = useTranslation('common');
  const { state, ask, reset } = useChatStream(subjectId);

  const isProcessing =
    state.status === 'thinking' ||
    state.status === 'searching' ||
    state.status === 'streaming';

  return (
    <div style={styles.root}>
      {/* ─── ページヘッダー ─── */}
      <header style={styles.header}>
        <h1 style={styles.title}>{t('qa.title')}</h1>
        <div style={styles.headerActions}>
          <span style={styles.subjectLabel}>{t('qa.subjectIdLabel')} {subjectId}</span>
          {state.status !== 'idle' && (
            <button
              onClick={reset}
              disabled={isProcessing}
              style={{
                ...styles.resetBtn,
                ...(isProcessing ? styles.resetBtnDisabled : {}),
              }}
              aria-label={t('qa.reset')}
            >
              {t('qa.reset')}
            </button>
          )}
        </div>
      </header>

      {/* ─── ストリーム表示エリア ─── */}
      <main style={styles.main}>
        {state.status === 'idle' ? (
          <div style={styles.emptyState}>
            <p style={styles.emptyStateText}>
              {t('qa.emptyStateMain')}
              <br />
              {t('qa.emptyStateSub')}
            </p>
          </div>
        ) : (
          <ChatStream state={state} />
        )}
      </main>

      {/* ─── 入力フォーム ─── */}
      <footer style={styles.footer}>
        <ChatInput onSubmit={ask} disabled={isProcessing} />
      </footer>
    </div>
  );
}

const styles = {
  root: {
    display: 'flex',
    flexDirection: 'column' as const,
    height: '100dvh',
    maxWidth: '800px',
    margin: '0 auto',
    background: '#ffffff',
    boxShadow: '0 0 0 1px #e0e0e0',
  },
  header: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: '12px 16px',
    borderBottom: '1px solid #e0e0e0',
    background: '#ffffff',
    flexShrink: 0,
  },
  title: {
    margin: 0,
    fontSize: '18px',
    fontWeight: 700 as const,
    color: '#212121',
  },
  headerActions: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
  },
  subjectLabel: {
    fontSize: '12px',
    color: '#9e9e9e',
    fontFamily: 'monospace',
  },
  resetBtn: {
    padding: '4px 12px',
    fontSize: '13px',
    color: '#616161',
    background: 'none',
    border: '1px solid #e0e0e0',
    borderRadius: '4px',
    cursor: 'pointer',
  },
  resetBtnDisabled: {
    opacity: 0.4,
    cursor: 'not-allowed',
  },
  main: {
    flex: 1,
    overflowY: 'auto' as const,
    display: 'flex',
    flexDirection: 'column' as const,
  },
  emptyState: {
    flex: 1,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    padding: '32px',
  },
  emptyStateText: {
    fontSize: '15px',
    color: '#9e9e9e',
    textAlign: 'center' as const,
    lineHeight: '1.8',
    margin: 0,
  },
  footer: {
    flexShrink: 0,
  },
} as const;
