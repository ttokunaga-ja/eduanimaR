/** SSE ストリームのステータス */
export type ChatStreamStatus =
  | 'idle'
  | 'thinking'
  | 'searching'
  | 'streaming'
  | 'done'
  | 'error'
  | 'clarification'; // 曖昧な質問への選択肢提示

/** エビデンスチャンク（根拠資料の断片） */
export interface EvidenceChunk {
  file_name: string;
  page_number: number;
  excerpt: string;
}

// ─── SSEイベント型 ────────────────────────────────────────────────────────────

export interface SSEThinkingEvent {
  type: 'thinking';
  data: Record<string, never>;
}

export interface SSESearchingEvent {
  type: 'searching';
  data: { query: string };
}

export interface SSEEvidenceEvent {
  type: 'evidence';
  data: { chunks: EvidenceChunk[] };
}

export interface SSEChunkEvent {
  type: 'chunk';
  data: { text: string };
}

export interface SSEDoneEvent {
  type: 'done';
  data: { chat_id: string; is_clarification?: boolean };
}

export interface SSEErrorEvent {
  type: 'error';
  data: { message: string };
}

/** 曖昧な質問への選択肢提示イベント */
export interface SSEClarificationEvent {
  type: 'clarification';
  data: {
    session_id: string;
    /** ユーザーに提示する具体的な質問候補（3〜5個） */
    options: string[];
  };
}

export type SSEEvent =
  | SSEThinkingEvent
  | SSESearchingEvent
  | SSEEvidenceEvent
  | SSEChunkEvent
  | SSEDoneEvent
  | SSEErrorEvent
  | SSEClarificationEvent;

// ─── フック戻り値の状態 ───────────────────────────────────────────────────────

export interface ChatStreamState {
  status: ChatStreamStatus;
  /** searching フェーズ中の検索クエリ */
  searchQuery?: string;
  /** evidence フェーズで選定された根拠チャンク */
  evidences: EvidenceChunk[];
  /** ストリーミング中の回答テキスト（逐次追記） */
  answer: string;
  /** done イベントで受け取った chat_id */
  chatId?: string;
  /** error イベントのメッセージ */
  error?: string;
  /** clarification フェーズで提示する質問候補（3〜5個） */
  clarificationOptions?: string[];
}
