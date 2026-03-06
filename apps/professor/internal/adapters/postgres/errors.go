package postgres

import (
	"database/sql"
	"errors"

	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/domain"
)

// mapDBError は database/sql レベルのエラーをドメインエラーに変換する。
// sql.ErrNoRows → domain.ErrNotFound
// その他のエラーはそのまま返す。
func mapDBError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}
