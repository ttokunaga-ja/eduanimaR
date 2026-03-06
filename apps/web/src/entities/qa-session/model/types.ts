import type { ChatSummary, ChatDetail, ChatDetailSourcesItem } from '@/shared/api';

/** QAセッションサマリー（一覧表示用） */
export type QASession = ChatSummary;

/** QAセッション詳細（過去チャット表示用） */
export type QASessionDetail = ChatDetail;

/** エビデンスソース（根拠資料） */
export type EvidenceSource = ChatDetailSourcesItem;
