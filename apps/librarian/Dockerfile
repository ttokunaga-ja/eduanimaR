# ─── ビルドステージ ───────────────────────────────────────────────
FROM python:3.12-slim AS builder

WORKDIR /build

# 依存インストール（キャッシュ最適化）
COPY pyproject.toml .
RUN pip install --upgrade pip && \
    pip install --no-cache-dir ".[dev]" && \
    pip install --no-cache-dir hatchling

# アプリケーションコードと生成済みprotoをコピー
COPY src/ src/
COPY gen/ gen/

# ─── ランタイムステージ ───────────────────────────────────────────
FROM python:3.12-slim AS runtime

WORKDIR /app

# ランタイム依存のみインストール
COPY pyproject.toml .
RUN pip install --upgrade pip && \
    pip install --no-cache-dir .

# ビルドステージからアプリケーションコードをコピー
COPY --from=builder /build/src/ src/
COPY --from=builder /build/gen/ gen/

ENV PYTHONPATH=/app/src
ENV PYTHONUNBUFFERED=1

# gRPC ポート
EXPOSE 50051
EXPOSE 8081

CMD ["python", "-m", "librarian.main"]
