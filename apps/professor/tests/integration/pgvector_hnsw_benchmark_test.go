//go:build integration

package integration

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	pgvector "github.com/pgvector/pgvector-go"
)

func BenchmarkChunksVectorSearch_HNSW(b *testing.B) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		b.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		b.Fatalf("connect db: %v", err)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, "SET hnsw.ef_search = 100"); err != nil {
		b.Fatalf("set hnsw.ef_search: %v", err)
	}

	query := randomVector(768)
	subjectID := os.Getenv("BENCH_SUBJECT_ID")
	if subjectID == "" {
		b.Skip("BENCH_SUBJECT_ID is not set")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		rows, err := conn.Query(ctx, `
			SELECT chunk_id, file_id, subject_id, content, page_number, chunk_index,
			       (1 - (embedding <=> $1::vector)) AS score
			FROM chunks
			WHERE subject_id = $2
			ORDER BY embedding <=> $1::vector
			LIMIT 8
		`, query, subjectID)
		if err != nil {
			b.Fatalf("query vector search: %v", err)
		}

		var count int
		for rows.Next() {
			count++
		}
		rows.Close()
		if count == 0 {
			b.Fatalf("no rows returned for subject_id=%s", subjectID)
		}

		elapsed := time.Since(start)
		b.ReportMetric(float64(elapsed.Microseconds()), "latency_us/op")
	}
}

func randomVector(dim int) pgvector.Vector {
	r := rand.New(rand.NewSource(42))
	v := make([]float32, dim)
	for i := 0; i < dim; i++ {
		v[i] = r.Float32()
	}
	return pgvector.NewVector(v)
}

func TestHNSWTuningExplainAnalyze(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is not set")
	}

	subjectID := os.Getenv("BENCH_SUBJECT_ID")
	if subjectID == "" {
		t.Skip("BENCH_SUBJECT_ID is not set")
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	defer conn.Close(ctx)

	query := randomVector(768)
	levels := []int{40, 80, 120}
	for _, ef := range levels {
		if _, err := conn.Exec(ctx, fmt.Sprintf("SET hnsw.ef_search = %d", ef)); err != nil {
			t.Fatalf("set ef_search=%d: %v", ef, err)
		}

		var plan string
		err := conn.QueryRow(ctx, `
			EXPLAIN (ANALYZE, FORMAT TEXT)
			SELECT chunk_id
			FROM chunks
			WHERE subject_id = $2
			ORDER BY embedding <=> $1::vector
			LIMIT 8
		`, query, subjectID).Scan(&plan)
		if err != nil {
			t.Fatalf("explain analyze ef_search=%d: %v", ef, err)
		}

		t.Logf("ef_search=%d plan=%s", ef, plan)
	}
}
