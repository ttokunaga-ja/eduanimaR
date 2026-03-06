package domain

import "errors"

// ドメイン共通エラー。アダプター層でこれらに変換し、HTTP層でステータスコードにマッピングする。
var (
	// ErrNotFound はリソースが存在しない場合
	ErrNotFound = errors.New("not found")
	// ErrForbidden はリソースへのアクセス権限がない場合
	ErrForbidden = errors.New("forbidden")
	// ErrInvalidInput は入力値が不正な場合
	ErrInvalidInput = errors.New("invalid input")
	// ErrConflict はリソースが既に存在する場合
	ErrConflict = errors.New("conflict")
)
