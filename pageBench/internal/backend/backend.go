// Package backend defines the RAGBackend interface and common types.
package backend

import "io"

// QueryResult は RAG バックエンドからのクエリ結果。
type QueryResult struct {
	Answer        string
	LatencyMS     int
	Sources       []Source
	LoopCount     int
	LibrarianMS   int
	AnswerGenMS   int
	Answerability string // "answerable" | "unanswerable" | "" (professor GenerateAnswerMeta から取得)
}

// Source はクエリ結果に含まれるソース参照。
type Source struct {
	Name string
	Page string
}

// RAGBackend は RAG システムの共通インターフェース。
type RAGBackend interface {
	// CreateCollection はドキュメントコレクション（ベクターストア等）を作成して ID を返す。
	CreateCollection(name string) (collectionID string, err error)

	// UploadDocument はコレクションにドキュメントをアップロードする。
	// chat_completions モードでは no-op。
	UploadDocument(collectionID string, name string, r io.Reader) (fileID string, err error)

	// WaitForReady はインデックス処理の完了を待機する。
	// chat_completions モードでは即時 true を返す。
	WaitForReady(collectionID string, timeoutSecs int, pollIntervalSecs int) (ready bool, err error)

	// Query はコレクションに対して質問を行い QueryResult を返す。
	Query(collectionID string, question string) (*QueryResult, error)

	// Cleanup はコレクションおよび関連リソースを削除する。
	// chat_completions モードでは no-op。
	Cleanup(collectionID string) error
}
