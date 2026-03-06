# ─── ビルドステージ ───────────────────────────────────────────────
FROM python:3.12-slim AS builder

WORKDIR /build

# 依存インストール（キャッシュ最適化）
COPY pyproject.toml .
RUN pip install --upgrade pip && \
    pip install --no-cache-dir ".[dev]" && \
    pip install --no-cache-dir hatchling

# Proto コード生成
COPY proto/ proto/
RUN mkdir -p src/librarian/v1 && \
    python -m grpc_tools.protoc \
        -I proto \
        --python_out=src \
        --grpc_python_out=src \
        --pyi_out=src \
        proto/librarian/v1/librarian.proto && \
    touch src/librarian/v1/__init__.py

# アプリケーションコードをコピー
COPY src/ src/

# ─── ランタイムステージ ───────────────────────────────────────────
FROM python:3.12-slim AS runtime

WORKDIR /app

# ランタイム依存のみインストール
COPY pyproject.toml .
RUN pip install --upgrade pip && \
    pip install --no-cache-dir .

# ビルドステージから生成済みコードをコピー
COPY --from=builder /build/src/ src/

ENV PYTHONPATH=/app/src
ENV PYTHONUNBUFFERED=1

# gRPC ポート
EXPOSE 50051

CMD ["python", "-m", "librarian.main"]
