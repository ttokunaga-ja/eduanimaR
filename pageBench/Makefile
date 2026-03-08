# pageBench Makefile

BINARY   := pagebench
CMD_DIR  := ./cmd/pagebench
BUILD_DIR := ./bin

.PHONY: all build test lint clean install help

## help: このヘルプを表示する
help:
	@echo "pageBench - RAG システム性能評価ツール"
	@echo ""
	@echo "使い方:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/  /'

## build: バイナリをビルドする (./bin/pagebench)
build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) $(CMD_DIR)
	@echo "✓ ビルド完了: $(BUILD_DIR)/$(BINARY)"

## install: バイナリを $GOPATH/bin にインストールする
install:
	go install $(CMD_DIR)
	@echo "✓ インストール完了: $$(go env GOPATH)/bin/$(BINARY)"

## test: ユニットテストを実行する
test:
	go test ./... -v -count=1

## test-metrics: metrics パッケージのテストのみ実行する
test-metrics:
	go test ./internal/metrics/... -v -count=1

## lint: go vet を実行する
lint:
	go vet ./...

## tidy: go mod tidy を実行する
tidy:
	go mod tidy

## clean: ビルド成果物を削除する
clean:
	rm -rf $(BUILD_DIR)

## check: 設定確認を実行する
check: build
	$(BUILD_DIR)/$(BINARY) check

## generate-sample: サンプルドメインで QA 生成を実行する
generate-sample: build
	$(BUILD_DIR)/$(BINARY) generate --domain ./domains/00_sample

## prepare-sample: サンプルドメインで Evaluation Preparation（ファイルアップロード + インデックス作成）を実行する
prepare-sample: build
	$(BUILD_DIR)/$(BINARY) prepare --domain ./domains/00_sample

## eval-sample: サンプルドメインで評価を実行する (judge スキップ)
eval-sample: build
	$(BUILD_DIR)/$(BINARY) eval --domain ./domains/00_sample --skip-judge

## all: tidy → lint → test → build
all: tidy lint test build
