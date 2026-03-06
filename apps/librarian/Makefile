.PHONY: install proto run dev lint typecheck test test-unit test-integration test-contract clean

# Python venv
VENV := .venv
PYTHON := $(VENV)/bin/python
DEPS_STAMP := $(VENV)/.deps-installed
GEN_PROTO_DIR := gen/proto/python/librarian/v1

## ─── 環境セットアップ ──────────────────────────────────────────────
install:
	@set -e; \
	if [ ! -x "$(PYTHON)" ] || ! "$(PYTHON)" -c "import sys" >/dev/null 2>&1; then \
		echo "==> Recreating Python virtual environment"; \
		rm -rf "$(VENV)"; \
		python3 -m venv "$(VENV)"; \
		rm -f "$(DEPS_STAMP)"; \
	fi; \
	if [ ! -f "$(DEPS_STAMP)" ] || [ pyproject.toml -nt "$(DEPS_STAMP)" ]; then \
		echo "==> Installing Python dependencies"; \
		"$(PYTHON)" -m ensurepip --upgrade; \
		"$(PYTHON)" -m pip install --upgrade pip; \
		"$(PYTHON)" -m pip install -e ".[dev]"; \
		touch "$(DEPS_STAMP)"; \
	fi

## ─── Proto コード生成 ─────────────────────────────────────────────
# contracts/proto/ が正本。grpcio-tools で Python stubs を生成し gen/ に配置する。
proto: install
	mkdir -p $(GEN_PROTO_DIR)
	$(PYTHON) -m grpc_tools.protoc \
		-I ../../contracts/proto \
		--python_out=gen/proto/python \
		--grpc_python_out=gen/proto/python \
		--pyi_out=gen/proto/python \
		../../contracts/proto/librarian/v1/librarian.proto
	@echo "✅ Proto stubs generated → $(GEN_PROTO_DIR)/"

## ─── 実行 ────────────────────────────────────────────────────────
run: proto
	PYTHONPATH=src $(PYTHON) -m librarian.main

dev: proto
	PYTHONPATH=src $(PYTHON) -m librarian.main

## ─── 品質チェック ────────────────────────────────────────────────
lint: install
	$(VENV)/bin/ruff check src/librarian tests/
	$(VENV)/bin/ruff format --check src/librarian tests/

typecheck: proto
	PYTHONPATH=src $(PYTHON) -m mypy src/librarian

format: install
	$(VENV)/bin/ruff format src/librarian tests/

test: test-unit

test-unit: proto
	PYTHONPATH=src $(PYTHON) -m pytest tests/ -v

test-integration: proto
	PYTHONPATH=src $(PYTHON) -m pytest tests/integration -v

test-contract: proto
	PYTHONPATH=src $(PYTHON) -m pytest tests/contract -v

## ─── クリーンアップ ──────────────────────────────────────────────
clean:
	rm -rf $(VENV) src/librarian/v1/__pycache__ __pycache__
	find . -name "*.pyc" -delete
