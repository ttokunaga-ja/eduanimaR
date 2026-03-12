from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class QuestionClarity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    QUESTION_CLARITY_UNSPECIFIED: _ClassVar[QuestionClarity]
    QUESTION_CLARITY_CLEAR: _ClassVar[QuestionClarity]
    QUESTION_CLARITY_AMBIGUOUS: _ClassVar[QuestionClarity]
QUESTION_CLARITY_UNSPECIFIED: QuestionClarity
QUESTION_CLARITY_CLEAR: QuestionClarity
QUESTION_CLARITY_AMBIGUOUS: QuestionClarity

class ThinkRequest(_message.Message):
    __slots__ = ("request_id", "user_query", "subject_id", "state", "search_history", "constraints")
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    USER_QUERY_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    SEARCH_HISTORY_FIELD_NUMBER: _ClassVar[int]
    CONSTRAINTS_FIELD_NUMBER: _ClassVar[int]
    request_id: str
    user_query: str
    subject_id: str
    state: str
    search_history: _containers.RepeatedCompositeFieldContainer[SearchHistory]
    constraints: Constraints
    def __init__(self, request_id: _Optional[str] = ..., user_query: _Optional[str] = ..., subject_id: _Optional[str] = ..., state: _Optional[str] = ..., search_history: _Optional[_Iterable[_Union[SearchHistory, _Mapping]]] = ..., constraints: _Optional[_Union[Constraints, _Mapping]] = ...) -> None: ...

class SearchHistory(_message.Message):
    __slots__ = ("step", "action", "queries_text", "rationale", "result_count")
    STEP_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    QUERIES_TEXT_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    RESULT_COUNT_FIELD_NUMBER: _ClassVar[int]
    step: int
    action: str
    queries_text: _containers.RepeatedScalarFieldContainer[str]
    rationale: str
    result_count: int
    def __init__(self, step: _Optional[int] = ..., action: _Optional[str] = ..., queries_text: _Optional[_Iterable[str]] = ..., rationale: _Optional[str] = ..., result_count: _Optional[int] = ...) -> None: ...

class Constraints(_message.Message):
    __slots__ = ("max_loops", "max_results", "timeout_ms", "thinking_level", "interpreted_query", "completion_criteria", "clarity")
    MAX_LOOPS_FIELD_NUMBER: _ClassVar[int]
    MAX_RESULTS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    THINKING_LEVEL_FIELD_NUMBER: _ClassVar[int]
    INTERPRETED_QUERY_FIELD_NUMBER: _ClassVar[int]
    COMPLETION_CRITERIA_FIELD_NUMBER: _ClassVar[int]
    CLARITY_FIELD_NUMBER: _ClassVar[int]
    max_loops: int
    max_results: int
    timeout_ms: int
    thinking_level: str
    interpreted_query: str
    completion_criteria: _containers.RepeatedScalarFieldContainer[str]
    clarity: QuestionClarity
    def __init__(self, max_loops: _Optional[int] = ..., max_results: _Optional[int] = ..., timeout_ms: _Optional[int] = ..., thinking_level: _Optional[str] = ..., interpreted_query: _Optional[str] = ..., completion_criteria: _Optional[_Iterable[str]] = ..., clarity: _Optional[_Union[QuestionClarity, str]] = ...) -> None: ...

class ThinkResponse(_message.Message):
    __slots__ = ("request_id", "search", "complete", "error")
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    SEARCH_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    request_id: str
    search: SearchAction
    complete: CompleteAction
    error: ErrorAction
    def __init__(self, request_id: _Optional[str] = ..., search: _Optional[_Union[SearchAction, _Mapping]] = ..., complete: _Optional[_Union[CompleteAction, _Mapping]] = ..., error: _Optional[_Union[ErrorAction, _Mapping]] = ...) -> None: ...

class SearchAction(_message.Message):
    __slots__ = ("queries_text", "queries_vector", "rationale", "exclude_chunk_ids")
    QUERIES_TEXT_FIELD_NUMBER: _ClassVar[int]
    QUERIES_VECTOR_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    EXCLUDE_CHUNK_IDS_FIELD_NUMBER: _ClassVar[int]
    queries_text: _containers.RepeatedScalarFieldContainer[str]
    queries_vector: _containers.RepeatedScalarFieldContainer[str]
    rationale: str
    exclude_chunk_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, queries_text: _Optional[_Iterable[str]] = ..., queries_vector: _Optional[_Iterable[str]] = ..., rationale: _Optional[str] = ..., exclude_chunk_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class CompleteAction(_message.Message):
    __slots__ = ("evidence", "coverage_notes")
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_NOTES_FIELD_NUMBER: _ClassVar[int]
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    coverage_notes: str
    def __init__(self, evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ..., coverage_notes: _Optional[str] = ...) -> None: ...

class Evidence(_message.Message):
    __slots__ = ("chunk_id", "why_relevant")
    CHUNK_ID_FIELD_NUMBER: _ClassVar[int]
    WHY_RELEVANT_FIELD_NUMBER: _ClassVar[int]
    chunk_id: str
    why_relevant: str
    def __init__(self, chunk_id: _Optional[str] = ..., why_relevant: _Optional[str] = ...) -> None: ...

class ErrorAction(_message.Message):
    __slots__ = ("error_type", "message")
    ERROR_TYPE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    error_type: str
    message: str
    def __init__(self, error_type: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...
