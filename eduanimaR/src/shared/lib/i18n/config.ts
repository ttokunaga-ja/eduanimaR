/**
 * i18next 設定（インライン翻訳リソース使用）
 *
 * Phase 1: inline resources でバンドル内に翻訳を含める。
 * Phase 2: next-i18next / next-intl への移行を検討。
 */
import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

const ja = {
  chat: {
    thinking: 'AI が戦略を立案中…',
    searching: '資料を検索中',
    searchQueryOpen: '「',
    searchQueryClose: '」',
    evidenceTitle: '📚 根拠資料 ({{count}} 件)',
    page: 'p.{{num}}',
    answerLabel: '回答',
    spinner: '⏳',
    cursor: '▌',
    feedbackPrompt: 'この回答は役立ちましたか？',
    thumbsUp: '👍',
    thumbsDown: '👎',
    errorLabel: 'エラー:',
    unknownError: '不明なエラーが発生しました',
  },
  qa: {
    title: '📖 資料に質問する',
    subjectIdLabel: '科目 ID:',
    reset: 'リセット',
    emptyStateMain: '下の入力欄から質問を入力してください。',
    emptyStateSub: 'アップロード済みの資料をもとに AI が回答します。',
  },
  input: {
    placeholder: '質問を入力してください（Enter で送信、Shift+Enter で改行）',
    submit: '送信',
    submitting: '処理中…',
  },
};

const en = {
  chat: {
    thinking: 'AI is planning strategy…',
    searching: 'Searching documents',
    searchQueryOpen: '"',
    searchQueryClose: '"',
    evidenceTitle: '📚 References ({{count}} items)',
    page: 'p.{{num}}',
    answerLabel: 'Answer',
    spinner: '⏳',
    cursor: '▌',
    feedbackPrompt: 'Was this answer helpful?',
    thumbsUp: '👍',
    thumbsDown: '👎',
    errorLabel: 'Error:',
    unknownError: 'An unknown error occurred',
  },
  qa: {
    title: '📖 Ask about documents',
    subjectIdLabel: 'Subject ID:',
    reset: 'Reset',
    emptyStateMain: 'Please enter your question in the input box below.',
    emptyStateSub: 'The AI will answer based on your uploaded documents.',
  },
  input: {
    placeholder: 'Enter your question (Enter to send, Shift+Enter for newline)',
    submit: 'Send',
    submitting: 'Processing…',
  },
};

if (!i18n.isInitialized) {
  i18n.use(initReactI18next).init({
    resources: {
      ja: { common: ja },
      en: { common: en },
    },
    lng: 'ja',
    fallbackLng: 'en',
    defaultNS: 'common',
    interpolation: {
      escapeValue: false, // React already escapes values
    },
  });
}

export default i18n;
