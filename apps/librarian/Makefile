.PHONY: install proto run dev lint test test-unit test-integration test-contract clean

# Python venv
VENV := .venv
PYTHON := $(VENV)/bin/python
DEPS_STAMP := $(VENV)/.deps-installed

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
# contracts/proto/ が正本。grpcio-tools で Python stubs を生成し src/ 以下に配置する
# 生成先: src/librarian/v1/librarian_pb2.py, librarian_pb2_grpc.py
proto: install
	mkdir -p src/librarian/v1
	$(PYTHON) -m grpc_tools.protoc \
		-I ../../contracts/proto \
		--python_out=src \
		--grpc_python_out=src \
		--pyi_out=src \
		../../contracts/proto/librarian/v1/librarian.proto
	# __init__.py を補完
	touch src/librarian/v1/__init__.py
	@echo "✅ Proto stubs generated → src/librarian/v1/"

## ─── 実行 ────────────────────────────────────────────────────────
run: install
	PYTHONPATH=src $(PYTHON) -m librarian.main

dev: install
	PYTHONPATH=src $(PYTHON) -m librarian.main

## ─── 品質チェック ────────────────────────────────────────────────
lint: install
	$(VENV)/bin/ruff check src/librarian tests/
	$(VENV)/bin/ruff format --check src/librarian tests/

format: install
	$(VENV)/bin/ruff format src/librarian tests/

test: test-unit

test-unit: install
	PYTHONPATH=src $(PYTHON) -m pytest tests/ -v

test-integration: install
	PYTHONPATH=src $(PYTHON) -m pytest tests/integration -v

test-contract: install
	PYTHONPATH=src $(PYTHON) -m pytest tests/contract -v

## ─── クリーンアップ ──────────────────────────────────────────────
clean:
	rm -rf $(VENV) src/librarian/v1/__pycache__ __pycache__
	find . -name "*.pyc" -delete
