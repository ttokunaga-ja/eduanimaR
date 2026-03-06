package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	pgvector "github.com/pgvector/pgvector-go"

	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/adapters/postgres/sqlcgen"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/domain"
	"github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor/internal/ports"
)

type chunkRepo struct {
	q *sqlcgen.Queries
}

// NewChunkRepo は ports.ChunkRepository の postgres 実装を返す。
func NewChunkRepo(db *sql.DB) ports.ChunkRepository {
	return &chunkRepo{q: sqlcgen.New(db)}
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
			SubjectID:  c.SubjectID,
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
// Note: sqlcgen の SearchChunksByVectorParams.Column1 は `$1::vector` に対応する。
func (r *chunkRepo) SearchByVector(ctx context.Context, subjectID uuid.UUID, embedding pgvector.Vector, limit int) ([]*domain.SearchResult, error) {
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

// SearchByText は PostgreSQL 全文検索（plainto_tsquery）を実行する。
func (r *chunkRepo) SearchByText(ctx context.Context, subjectID uuid.UUID, query string, limit int) ([]*domain.SearchResult, error) {
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

func (r *chunkRepo) DeleteByFileID(ctx context.Context, fileID uuid.UUID) error {
	return r.q.DeleteChunksByFileID(ctx, fileID)
}

// ─── 変換ヘルパー ─────────────────────────────────────────────────

func sqlcChunkToDomainChunk(row sqlcgen.Chunk) *domain.Chunk {
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
