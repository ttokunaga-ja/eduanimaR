// Package domain はビジネスロジックのコアエンティティを定義する。
// 外部パッケージ（DB/HTTP/gRPC）に依存しない。
package domain

import (
	"time"

	"github.com/google/uuid"
)

// DevUserID は Phase 1 の固定開発ユーザー UUID
var DevUserID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// DevUserEmail は Phase 1 の固定開発ユーザーメールアドレス
const DevUserEmail = "dev@example.com"

// User はユーザーエンティティ
type User struct {
	ID             uuid.UUID
	Email          string
	Provider       *string // nullable（Phase 2 で SSO 実装時に使用）
	ProviderUserID *string // nullable（Phase 2 で SSO 実装時に使用）
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
