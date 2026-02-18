# Makefile — eduanima-professor
# 使い方: make help

.PHONY: all proto sqlc generate migrate migrate-dry migrate-hash \
        build run dev test test-unit test-integration test-contract lint fmt clean help

MODULE  = github.com/ttokunaga-ja/eduanimaR/eduanimaR_Professor
BINARY  = bin/eduanima-professor
DATABASE_URL ?= postgres://eduanima:eduanima_password@localhost:5432/eduanima_professor?sslmode=disable

# ─────────────────────────────────────────
# コード生成
# ─────────────────────────────────────────

## proto: .proto → Go コードを生成 (requires: buf)
proto:
	@echo "==> Generating gRPC/protobuf code..."
	buf generate
	@echo "==> Done: gen/proto/"

## sqlc: SQL → Go コードを生成 (requires: sqlc)
sqlc:
	@echo "==> Generating DB code from SQL..."
	sqlc generate
	@echo "==> Done: internal/adapters/postgres/sqlcgen/"

## generate: proto + sqlc を実行
generate: proto sqlc

# ─────────────────────────────────────────
# DB マイグレーション (Atlas)
# ─────────────────────────────────────────

## migrate: 未適用マイグレーションを DATABASE_URL に適用
migrate:
	@echo "==> Applying migrations to: $(DATABASE_URL)"
	atlas migrate apply \
		--dir "file://schema/migrations" \
		--url "$(DATABASE_URL)"
	@echo "==> Migration complete."

## migrate-dry: 未適用マイグレーションを表示（適用しない）
migrate-dry:
	atlas migrate status \
		--dir "file://schema/migrations" \
		--url "$(DATABASE_URL)"

## migrate-hash: atlas.sum を再計算（SQL ファイルを手動編集した後に実行）
migrate-hash:
	atlas migrate hash --dir "file://schema/migrations"

# ─────────────────────────────────────────
# ビルド / 実行
# ─────────────────────────────────────────

## build: バイナリをコンパイル
build:
	@echo "==> Building $(BINARY)..."
	go build -o $(BINARY) ./cmd/eduanima-professor
	@echo "==> $(BINARY) ready."

## run: ローカルで実行（環境変数 or .env が必要）
run:
	go run ./cmd/eduanima-professor/...

## dev: ホットリロードで実行 (requires: air)
dev:
	air -c .air.toml

# ─────────────────────────────────────────
# テスト
# ─────────────────────────────────────────

## test: 全テスト（ユニット）を実行
test:
	go test ./... -v -race -count=1

## test-unit: UseCase ユニットテストのみ実行（外部依存不要）
test-unit:
	go test ./internal/usecases/... ./internal/domain/... -v -race -count=1

## test-integration: 統合テストを実行（DB 起動が必要）
test-integration:
	go test ./... -v -race -count=1 -tags=integration

## test-contract: コントラクトテストを実行
test-contract:
	go test ./internal/contracttest/... -v -race -count=1 -tags=contract

# ─────────────────────────────────────────
# Lint / フォーマット
# ─────────────────────────────────────────

## lint: golangci-lint を実行
lint:
	golangci-lint run ./...

## fmt: Go コードをフォーマット
fmt:
	gofmt -w .

# ─────────────────────────────────────────
# クリーン
# ─────────────────────────────────────────

## clean: ビルド成果物を削除
clean:
	rm -rf bin/

## help: 使用可能なターゲットを表示
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
