// Public API for API clients/hooks.
// Consumers should import from '@/shared/api', not from generated subdirectories.

export { apiFetch } from './client';
export { ApiError, isApiError } from './errors';

// ─── Professor API (generated) ───────────────────────────────────────────────
// Re-export the full generated API so consumers don't need to deep-import
// @/shared/api/generated/** (which violates no-restricted-imports).
export {
  // Chat
  postV1SubjectsSubjectIdChats,
  getV1SubjectsSubjectIdChats,
  getV1SubjectsSubjectIdChatsChatId,
  postV1SubjectsSubjectIdChatsChatIdFeedback,
  // Subjects
  getV1Subjects,
  postV1Subjects,
  getV1SubjectsSubjectId,
  deleteV1SubjectsSubjectId,
  // Materials
  getV1SubjectsSubjectIdMaterials,
  postV1SubjectsSubjectIdMaterials,
  getV1SubjectsSubjectIdMaterialsMaterialId,
  deleteV1SubjectsSubjectIdMaterialsMaterialId,
  // Auth (Phase 1)
  postV1AuthDevLogin,
  // Health
  getHealthz,
  getReadyz,
} from './generated/eduanimaProfessorAPI';

// ─── Generated model types ────────────────────────────────────────────────────
export type {
  ChatSummary,
  ChatDetail,
  ChatDetailSourcesItem,
  ChatDetailFeedback,
  Subject,
  Material,
  MaterialStatus,
  ErrorResponse,
  BadRequestResponse,
  NotFoundResponse,
  PostV1SubjectsSubjectIdChatsBody,
  PostV1SubjectsSubjectIdChatsChatIdFeedbackBody,
  GetV1SubjectsSubjectIdChats200,
  GetV1SubjectsSubjectIdChatsParams,
} from './generated/model';

// Const enums (not type-only)
export {
  PostV1SubjectsSubjectIdChatsChatIdFeedbackBodyRating,
  MaterialStatus as MaterialStatusEnum,
} from './generated/model';
