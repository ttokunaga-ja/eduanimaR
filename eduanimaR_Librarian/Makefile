.PHONY: install proto run dev lint test clean

# Python venv
VENV := .venv
PYTHON := $(VENV)/bin/python
PIP := $(VENV)/bin/pip

## ─── 環境セットアップ ──────────────────────────────────────────────
install: $(VENV)/bin/activate

$(VENV)/bin/activate: pyproject.toml
	python3 -m venv $(VENV)
	$(PIP) install --upgrade pip
	$(PIP) install -e ".[dev]"
	touch $@

## ─── Proto コード生成 ─────────────────────────────────────────────
# grpcio-tools で Python stubs を生成し src/ 以下に配置する
# 生成先: src/librarian/v1/librarian_pb2.py, librarian_pb2_grpc.py
proto: install
	mkdir -p src/librarian/v1
	$(PYTHON) -m grpc_tools.protoc \
		-I proto \
		--python_out=src \
		--grpc_python_out=src \
		--pyi_out=src \
		proto/librarian/v1/librarian.proto
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
	$(VENV)/bin/ruff check src/ tests/
	$(VENV)/bin/ruff format --check src/ tests/

format: install
	$(VENV)/bin/ruff format src/ tests/

test: install
	PYTHONPATH=src $(VENV)/bin/pytest tests/ -v

## ─── クリーンアップ ──────────────────────────────────────────────
clean:
	rm -rf $(VENV) src/librarian/v1/__pycache__ __pycache__
	find . -name "*.pyc" -delete
