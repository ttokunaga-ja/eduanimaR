# ─── 開発ステージ（ホットリロード: Air使用）────────────────────────
FROM golang:1.25-alpine AS dev

# build + air に必要なツール
RUN apk add --no-cache git ca-certificates tzdata curl && \
    go install github.com/air-verse/air@latest

WORKDIR /app

# 依存取得（キャッシュ最適化）
COPY go.mod go.sum ./
RUN go mod download

# ソースはボリュームマウントするので COPY しない
EXPOSE 8080

CMD ["air", "-c", ".air.toml"]

# ─── ビルドステージ ───────────────────────────────────────────────
FROM golang:1.25-alpine AS build

# build に必要なツール
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# 依存取得（キャッシュ最適化: go.mod/go.sum のみ先にコピー）
COPY go.mod go.sum ./
RUN go mod download

# ソースコード全体をコピー
COPY . .

# 静的リンクバイナリをビルド
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" \
    -o /professor \
    ./cmd/professor

# ─── プロダクションステージ ──────────────────────────────────────
# distroless: シェルなし・最小サーフェス
FROM gcr.io/distroless/static-debian12 AS production

WORKDIR /app

# CA証明書・タイムゾーンデータをビルドステージから持ち込む
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo

# バイナリのみコピー
COPY --from=build /professor /app/professor

EXPOSE 8080

ENTRYPOINT ["/app/professor"]
