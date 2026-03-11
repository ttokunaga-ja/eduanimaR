package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	pgvector "github.com/pgvector/pgvector-go"

	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/adapters/postgres/sqlcgen"
	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/apps/professor/internal/ports"
)

type chunkRepo struct {
	q  *sqlcgen.Queries
	db *sql.DB // 既読チャンク除外の動的クエリ用（excludeIDs が非空の場合）
}

// NewChunkRepo は ports.ChunkRepository の postgres 実装を返す。
func NewChunkRepo(db *sql.DB) ports.ChunkRepository {
	return &chunkRepo{q: sqlcgen.New(db), db: db}
}

func (r *chunkRepo) ListByFileID(ctx context.Context, fileID uuid.UUID) ([]*domain.Chunk, error) {
	rows, err := r.q.ListChunksByFileID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Chunk, len(rows))
	for i, row := range rows {
		result[i] = sqlcChunkToDomainChunk(row)
	}
	return result, nil
}

func (r *chunkRepo) BatchCreate(ctx context.Context, chunks []*domain.Chunk) error {
	for _, c := range chunks {
		var pageNum sql.NullInt32
		if c.PageNumber != nil {
			pageNum = sql.NullInt32{Int32: int32(*c.PageNumber), Valid: true}
		}
		_, err := r.q.InsertChunk(ctx, sqlcgen.InsertChunkParams{
			ChunkID:    c.ID,
			FileID:     c.FileID,
			PageNumber: pageNum,
			ChunkIndex: int32(c.ChunkIndex),
			Content:    c.Content,
			Embedding:  c.Embedding,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// SearchByVector は pgvector HNSW コサイン類似度検索を実行する。
// excludeIDs が空の場合は既存の sqlc クエリを使用（高速パス）。
// excludeIDs が非空の場合は動的 SQL で既読チャンクを DB レベルで除外する。
func (r *chunkRepo) SearchByVector(ctx context.Context, subjectID uuid.UUID, embedding pgvector.Vector, limit int, excludeIDs []uuid.UUID) ([]*domain.SearchResult, error) {
	if len(excludeIDs) == 0 {
		// 高速パス: sqlc 生成クエリ（除外なし）
		rows, err := r.q.SearchChunksByVector(ctx, sqlcgen.SearchChunksByVectorParams{
			Column1:   embedding,
			SubjectID: subjectID,
			Limit:     int32(limit),
		})
		if err != nil {
			return nil, err
		}
		result := make([]*domain.SearchResult, len(rows))
		for i, row := range rows {
			result[i] = sqlcVectorRowToSearchResult(row)
		}
		return result, nil
	}
	return r.searchByVectorWithExcl(ctx, subjectID, embedding, limit, excludeIDs)
}

// searchByVectorWithExcl は excludeIDs が非空の場合の動的クエリ実装。
// 各 UUID を個別のパラメータとして渡すことで、ドライバー依存の配列処理を回避する。
// UUID は事前に検証済みの型のため SQL インジェクションリスクはない。
func (r *chunkRepo) searchByVectorWithExcl(ctx context.Context, subjectID uuid.UUID, embedding pgvector.Vector, limit int, excludeIDs []uuid.UUID) ([]*domain.SearchResult, error) {
	// $1=embedding, $2=subjectID, $3=limit, $4..$N=excludeIDs
	args := make([]interface{}, 0, 3+len(excludeIDs))
	args = append(args, embedding, subjectID, int32(limit))

	placeholders := make([]string, len(excludeIDs))
	for i, id := range excludeIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+4)
		args = append(args, id)
	}

	query := `SELECT
  m.id AS chunk_id,
  m.raw_file_id AS file_id,
  rf.subject_id,
  rf.original_filename AS file_name,
  m.page_start AS page_number,
  m.sequence_in_file AS chunk_index,
  m.content_markdown AS content,
  m.created_at
FROM materials m
JOIN raw_files rf ON rf.id = m.raw_file_id
WHERE rf.subject_id = $2
  AND rf.is_active = TRUE
  AND m.is_active = TRUE
  AND m.id NOT IN (` + strings.Join(placeholders, ", ") + `)
ORDER BY m.embedding <=> $1::vector
LIMIT $3`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSearchResultRows(rows)
}

// SearchByText は PostgreSQL 全文検索（plainto_tsquery）を実行する。
// excludeIDs が空の場合は既存の sqlc クエリを使用（高速パス）。
// excludeIDs が非空の場合は動的 SQL で既読チャンクを DB レベルで除外する。
func (r *chunkRepo) SearchByText(ctx context.Context, subjectID uuid.UUID, query string, limit int, excludeIDs []uuid.UUID) ([]*domain.SearchResult, error) {
	if len(excludeIDs) == 0 {
		// 高速パス: sqlc 生成クエリ（除外なし）
		rows, err := r.q.SearchChunksByText(ctx, sqlcgen.SearchChunksByTextParams{
			PlaintoTsquery: query,
			SubjectID:      subjectID,
			Limit:          int32(limit),
		})
		if err != nil {
			return nil, err
		}
		result := make([]*domain.SearchResult, len(rows))
		for i, row := range rows {
			result[i] = sqlcTextRowToSearchResult(row)
		}
		return result, nil
	}
	return r.searchByTextWithExcl(ctx, subjectID, query, limit, excludeIDs)
}

// searchByTextWithExcl は excludeIDs が非空の場合の動的クエリ実装。
func (r *chunkRepo) searchByTextWithExcl(ctx context.Context, subjectID uuid.UUID, queryStr string, limit int, excludeIDs []uuid.UUID) ([]*domain.SearchResult, error) {
	// $1=query, $2=subjectID, $3=limit, $4..$N=excludeIDs
	args := make([]interface{}, 0, 3+len(excludeIDs))
	args = append(args, queryStr, subjectID, int32(limit))

	placeholders := make([]string, len(excludeIDs))
	for i, id := range excludeIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+4)
		args = append(args, id)
	}

	sqlQuery := `SELECT
  m.id AS chunk_id,
  m.raw_file_id AS file_id,
  rf.subject_id,
  rf.original_filename AS file_name,
  m.page_start AS page_number,
  m.sequence_in_file AS chunk_index,
  m.content_markdown AS content,
  m.created_at
FROM materials m
JOIN raw_files rf ON rf.id = m.raw_file_id
WHERE rf.subject_id = $2
  AND rf.is_active = TRUE
  AND m.is_active = TRUE
  AND to_tsvector('simple', m.content_markdown) @@ plainto_tsquery('simple', $1)
  AND m.id NOT IN (` + strings.Join(placeholders, ", ") + `)
ORDER BY ts_rank(to_tsvector('simple', m.content_markdown), plainto_tsquery('simple', $1)) DESC
LIMIT $3`

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSearchResultRows(rows)
}

// scanSearchResultRows は *sql.Rows を []domain.SearchResult に変換する共通ヘルパー。
// searchByVectorWithExcl / searchByTextWithExcl の両方で使用する。
func scanSearchResultRows(rows *sql.Rows) ([]*domain.SearchResult, error) {
	var results []*domain.SearchResult
	for rows.Next() {
		var (
			chunkID    uuid.UUID
			fileID     uuid.UUID
			subjectID  uuid.UUID
			fileName   string
			pageNumber sql.NullInt32
			chunkIndex int32
			content    string
			createdAt  time.Time
		)
		if err := rows.Scan(
			&chunkID,
			&fileID,
			&subjectID,
			&fileName,
			&pageNumber,
			&chunkIndex,
			&content,
			&createdAt,
		); err != nil {
			return nil, err
		}
		sr := &domain.SearchResult{
			ChunkID:    chunkID,
			FileID:     fileID,
			SubjectID:  subjectID,
			FileName:   fileName,
			ChunkIndex: int(chunkIndex),
			Content:    content,
			CreatedAt:  createdAt,
		}
		if pageNumber.Valid {
			v := int(pageNumber.Int32)
			sr.PageNumber = &v
		}
		results = append(results, sr)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func (r *chunkRepo) DeleteByFileID(ctx context.Context, fileID uuid.UUID) error {
	return r.q.DeleteChunksByFileID(ctx, fileID)
}

// ─── 変換ヘルパー ─────────────────────────────────────────────────

func sqlcChunkToDomainChunk(row sqlcgen.ListChunksByFileIDRow) *domain.Chunk {
	c := &domain.Chunk{
		ID:         row.ChunkID,
		FileID:     row.FileID,
		SubjectID:  row.SubjectID,
		ChunkIndex: int(row.ChunkIndex),
		Content:    row.Content,
		Embedding:  row.Embedding,
		CreatedAt:  row.CreatedAt,
	}
	if row.PageNumber.Valid {
		v := int(row.PageNumber.Int32)
		c.PageNumber = &v
	}
	return c
}

func sqlcVectorRowToSearchResult(row sqlcgen.SearchChunksByVectorRow) *domain.SearchResult {
	sr := &domain.SearchResult{
		ChunkID:    row.ChunkID,
		FileID:     row.FileID,
		SubjectID:  row.SubjectID,
		FileName:   row.FileName,
		ChunkIndex: int(row.ChunkIndex),
		Content:    row.Content,
		CreatedAt:  row.CreatedAt,
	}
	if row.PageNumber.Valid {
		v := int(row.PageNumber.Int32)
		sr.PageNumber = &v
	}
	return sr
}

func sqlcTextRowToSearchResult(row sqlcgen.SearchChunksByTextRow) *domain.SearchResult {
	sr := &domain.SearchResult{
		ChunkID:    row.ChunkID,
		FileID:     row.FileID,
		SubjectID:  row.SubjectID,
		FileName:   row.FileName,
		ChunkIndex: int(row.ChunkIndex),
		Content:    row.Content,
		CreatedAt:  row.CreatedAt,
	}
	if row.PageNumber.Valid {
		v := int(row.PageNumber.Int32)
		sr.PageNumber = &v
	}
	return sr
}
