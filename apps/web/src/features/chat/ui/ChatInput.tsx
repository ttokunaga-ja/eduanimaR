'use client';

import { useState, type FormEvent, type KeyboardEvent } from 'react';

interface ChatInputProps {
  onSubmit: (question: string) => void;
  disabled?: boolean;
}

/**
 * 質問入力フォーム。
 * Shift+Enter で改行、Enter で送信。
 */
export function ChatInput({ onSubmit, disabled = false }: ChatInputProps) {
  const [value, setValue] = useState('');

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    const trimmed = value.trim();
    if (!trimmed || disabled) return;
    onSubmit(trimmed);
    setValue('');
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      const trimmed = value.trim();
      if (!trimmed || disabled) return;
      onSubmit(trimmed);
      setValue('');
    }
  };

  return (
    <form onSubmit={handleSubmit} style={styles.form}>
      <textarea
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="質問を入力してください（Enter で送信、Shift+Enter で改行）"
        disabled={disabled}
        rows={3}
        style={styles.textarea}
        aria-label="質問入力"
      />
      <button
        type="submit"
        disabled={disabled || !value.trim()}
        style={{
          ...styles.button,
          ...(disabled || !value.trim() ? styles.buttonDisabled : {}),
        }}
        aria-label="送信"
      >
        {disabled ? '処理中…' : '送信'}
      </button>
    </form>
  );
}

const styles = {
  form: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '8px',
    padding: '16px',
    borderTop: '1px solid #e0e0e0',
    background: '#fafafa',
  },
  textarea: {
    width: '100%',
    padding: '10px 12px',
    fontSize: '14px',
    lineHeight: '1.5',
    border: '1px solid #d0d0d0',
    borderRadius: '6px',
    resize: 'vertical' as const,
    outline: 'none',
    fontFamily: 'inherit',
    boxSizing: 'border-box' as const,
  },
  button: {
    alignSelf: 'flex-end',
    padding: '8px 24px',
    fontSize: '14px',
    fontWeight: 600,
    color: '#ffffff',
    background: '#1976d2',
    border: 'none',
    borderRadius: '6px',
    cursor: 'pointer',
  },
  buttonDisabled: {
    background: '#b0bec5',
    cursor: 'not-allowed',
  },
} as const;
