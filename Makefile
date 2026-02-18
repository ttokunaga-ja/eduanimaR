# ============================================================
# eduanimaR — ルート Makefile
#
# すべての操作は Docker Compose 経由で管理する。
# ローカルに Go / Python / Node をインストールしなくても開発できる。
#
# 使い方: make help
# ============================================================

.PHONY: help infra dev prod down restart logs ps \
        migrate migrate-dry migrate-hash build build-prod \
        test-all test-professor test-librarian test-frontend \
        lint-all clean \
        deploy deploy-librarian deploy-professor deploy-frontend

# ─────────────────────────────────────────────────────────────
# 定数
# ─────────────────────────────────────────────────────────────
COMPOSE         := docker compose
COMPOSE_PROD    := docker compose -f docker-compose.yml -f docker-compose.prod.yml

# ─────────────────────────────────────────────────────────────
# セットアップ
# ─────────────────────────────────────────────────────────────

## setup: .env を作成する（初回のみ）
setup:
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "✅ .env を作成しました。GEMINI_API_KEY を設定してください: vim .env"; \
	else \
		echo "ℹ️  .env はすでに存在します。スキップしました。"; \
	fi

# ─────────────────────────────────────────────────────────────
# 起動（開発）
# ─────────────────────────────────────────────────────────────

## infra: インフラのみ起動（postgres / minio / kafka）
infra:
	@echo "==> インフラ起動中..."
	$(COMPOSE) up postgres minio minio-init kafka kafka-init -d
	@echo "✅ インフラ起動完了"
	@echo "   PostgreSQL : localhost:5432"
	@echo "   MinIO API  : localhost:9000"
	@echo "   MinIO UI   : http://localhost:9001  (admin: minioadmin)"
	@echo "   Kafka      : localhost:9094"

## dev: 全サービス起動（開発モード・ホットリロード）
dev: setup
	@echo "==> 開発環境を起動中..."
	$(COMPOSE) up --build
# ※ -d を付けるとデタッチモード（バックグラウンド）で起動する

## dev-d: 全サービスをバックグラウンドで起動
dev-d: setup
	$(COMPOSE) up --build -d
	@echo "✅ バックグラウンド起動完了"
	@echo "   Frontend  : http://localhost:3000"
	@echo "   Professor : http://localhost:8080"
	@echo "   Librarian : grpc://localhost:50051"
	@echo "   MinIO UI  : http://localhost:9001"
	@$(MAKE) logs

# ─────────────────────────────────────────────────────────────
# 起動（プロダクション）
# ─────────────────────────────────────────────────────────────

## prod: プロダクションビルドで全サービス起動
prod: setup
	@echo "==> プロダクションビルドを開始します..."
	$(COMPOSE_PROD) up --build -d
	@echo "✅ プロダクション起動完了"

# ─────────────────────────────────────────────────────────────
# 停止 / 再起動
# ─────────────────────────────────────────────────────────────

## down: 全コンテナ停止（ボリューム保持）
down:
	$(COMPOSE) down

## restart: 全コンテナ再起動
restart:
	$(COMPOSE) restart

# ─────────────────────────────────────────────────────────────
# ログ / 状態確認
# ─────────────────────────────────────────────────────────────

## logs: 全サービスのログを追尾
logs:
	$(COMPOSE) logs -f

## logs-professor: Professor のログのみ追尾
logs-professor:
	$(COMPOSE) logs -f professor

## logs-librarian: Librarian のログのみ追尾
logs-librarian:
	$(COMPOSE) logs -f librarian

## logs-frontend: Frontend のログのみ追尾
logs-frontend:
	$(COMPOSE) logs -f frontend

## ps: コンテナの状態を確認
ps:
	$(COMPOSE) ps

# ─────────────────────────────────────────────────────────────
# DB マイグレーション
# ─────────────────────────────────────────────────────────────

## migrate: DB マイグレーションを適用（postgres 起動後に実行）
migrate:
	@echo "==> マイグレーション適用中..."
	cd eduanimaR_Professor && make migrate
	@echo "✅ マイグレーション完了"

## migrate-dry: 未適用マイグレーションを確認（適用しない）
migrate-dry:
	cd eduanimaR_Professor && make migrate-dry

## migrate-hash: migration ファイル変更後の Atlas ハッシュを再生成（必須）
## ⚠️  migration ファイルを直接編集した後は必ずこのコマンドを実行すること
## 　  実行しないと atlas migrate apply が "ERR: checksum mismatch" で失敗する
migrate-hash:
	@echo "==> Atlas migrate hash 再生成中..."
	cd eduanimaR_Professor && atlas migrate hash --dir "file://schema/migrations"
	@echo "✅ atlas.sum を更新しました"
	@echo "   git add eduanimaR_Professor/schema/migrations/atlas.sum でコミットしてください"

# ─────────────────────────────────────────────────────────────
# ビルド
# ─────────────────────────────────────────────────────────────

## build: 全サービスの Docker イメージをビルド（dev）
build:
	$(COMPOSE) build

## build-prod: 全サービスの Docker イメージをビルド（production）
build-prod:
	$(COMPOSE_PROD) build

# ─────────────────────────────────────────────────────────────
# テスト（コンテナ外で実行）
# ─────────────────────────────────────────────────────────────

## test-all: 全サービスのユニットテストを実行
test-all: test-professor test-librarian test-frontend
	@echo "✅ 全テスト完了"

## test-professor: Go ユニットテスト（外部依存不要）
test-professor:
	@echo "==> [Professor] Go ユニットテスト..."
	cd eduanimaR_Professor && make test-unit

## test-librarian: Python ユニットテスト
test-librarian:
	@echo "==> [Librarian] Python ユニットテスト..."
	cd eduanimaR_Librarian && make test

## test-frontend: フロントエンド ユニットテスト
test-frontend:
	@echo "==> [Frontend] Vitest ユニットテスト..."
	cd eduanimaR && npm run test

## test-e2e: E2E テスト（dev サーバーが起動している状態で実行）
test-e2e:
	@echo "==> [Frontend] Playwright E2E テスト..."
	cd eduanimaR && npm run test:e2e

# ─────────────────────────────────────────────────────────────
# Lint
# ─────────────────────────────────────────────────────────────

## lint-all: 全サービスの Lint を実行
lint-all:
	@echo "==> [Professor] golangci-lint..."
	cd eduanimaR_Professor && make lint
	@echo "==> [Librarian] ruff..."
	cd eduanimaR_Librarian && make lint
	@echo "==> [Frontend] ESLint..."
	cd eduanimaR && npm run lint

# ─────────────────────────────────────────────────────────────
# クリーン
# ─────────────────────────────────────────────────────────────

## clean: コンテナ・ボリューム・ネットワークをすべて削除
clean:
	@echo "⚠️  DB データ含む全ボリュームを削除します..."
	$(COMPOSE) down -v --remove-orphans
	@echo "✅ クリーン完了"

## clean-images: ビルドイメージも削除
clean-images: clean
	docker rmi $$(docker images "eduanimar*" -q) 2>/dev/null || true
	@echo "✅ イメージ削除完了"

# ─────────────────────────────────────────────────────────────
# Cloud Run デプロイ（本番）
# 使い方: make deploy PROJECT_ID=your-gcp-project-id
# 詳細: CLOUD_RUN.md 参照
# ─────────────────────────────────────────────────────────────
PROJECT_ID   ?= $(shell gcloud config get-value project 2>/dev/null)
REGION       ?= asia-northeast1
REPO         ?= eduanima
SHORT_SHA    ?= $(shell git rev-parse --short HEAD)

## deploy: 全サービスを Cloud Run にビルド＆デプロイ（Cloud Build 使用）
deploy:
	@[ -n "$(PROJECT_ID)" ] || (echo "❌ PROJECT_ID を指定してください: make deploy PROJECT_ID=xxx"; exit 1)
	@echo "==> Cloud Build でビルド＆デプロイ中... (project: $(PROJECT_ID))"
	gcloud builds submit \
		--config cloudbuild.yaml \
		--project=$(PROJECT_ID) \
		--substitutions=_REGION=$(REGION),_REPO=$(REPO) \
		.

## deploy-librarian: Librarian のみ Cloud Run にデプロイ
deploy-librarian:
	@[ -n "$(PROJECT_ID)" ] || (echo "❌ PROJECT_ID を指定してください"; exit 1)
	@echo "==> [Librarian] Cloud Run デプロイ中..."
	docker build --target=runtime \
		-t $(REGION)-docker.pkg.dev/$(PROJECT_ID)/$(REPO)/eduanima-librarian:$(SHORT_SHA) \
		./eduanimaR_Librarian
	docker push $(REGION)-docker.pkg.dev/$(PROJECT_ID)/$(REPO)/eduanima-librarian:$(SHORT_SHA)
	gcloud run deploy eduanima-librarian \
		--image=$(REGION)-docker.pkg.dev/$(PROJECT_ID)/$(REPO)/eduanima-librarian:$(SHORT_SHA) \
		--region=$(REGION) \
		--ingress=internal \
		--project=$(PROJECT_ID)
	@echo "✅ Librarian デプロイ完了"

## deploy-professor: Professor のみ Cloud Run にデプロイ
deploy-professor:
	@[ -n "$(PROJECT_ID)" ] || (echo "❌ PROJECT_ID を指定してください"; exit 1)
	@echo "==> [Professor] Cloud Run デプロイ中..."
	docker build --target=production \
		-t $(REGION)-docker.pkg.dev/$(PROJECT_ID)/$(REPO)/eduanima-professor:$(SHORT_SHA) \
		./eduanimaR_Professor
	docker push $(REGION)-docker.pkg.dev/$(PROJECT_ID)/$(REPO)/eduanima-professor:$(SHORT_SHA)
	gcloud run deploy eduanima-professor \
		--image=$(REGION)-docker.pkg.dev/$(PROJECT_ID)/$(REPO)/eduanima-professor:$(SHORT_SHA) \
		--region=$(REGION) \
		--ingress=internal-and-cloud-load-balancing \
		--project=$(PROJECT_ID)
	@echo "✅ Professor デプロイ完了"

## deploy-frontend: Frontend のみ Cloud Run にデプロイ
deploy-frontend:
	@[ -n "$(PROJECT_ID)" ] || (echo "❌ PROJECT_ID を指定してください"; exit 1)
	@echo "==> [Frontend] Cloud Run デプロイ中..."
	docker build --target=production \
		-t $(REGION)-docker.pkg.dev/$(PROJECT_ID)/$(REPO)/eduanima-frontend:$(SHORT_SHA) \
		./eduanimaR
	docker push $(REGION)-docker.pkg.dev/$(PROJECT_ID)/$(REPO)/eduanima-frontend:$(SHORT_SHA)
	gcloud run deploy eduanima-frontend \
		--image=$(REGION)-docker.pkg.dev/$(PROJECT_ID)/$(REPO)/eduanima-frontend:$(SHORT_SHA) \
		--region=$(REGION) \
		--ingress=internal-and-cloud-load-balancing \
		--project=$(PROJECT_ID)
	@echo "✅ Frontend デプロイ完了"

# ─────────────────────────────────────────────────────────────
# ヘルプ
# ─────────────────────────────────────────────────────────────

## help: 使用可能なコマンド一覧を表示
help:
	@echo ""
	@echo "eduanimaR — Docker Compose 管理コマンド"
	@echo "==========================================="
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  make /'
	@echo ""
