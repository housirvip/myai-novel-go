### Stage 1: builder
# 注:用 alpine 的 Go 镜像 + CGO_ENABLED=1 是因为 mattn/go-sqlite3 需要 cgo;
# 如果 build 阶段切到 modernc 纯 Go sqlite 驱动,可改回 distroless static。
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache build-base git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1 GOOS=linux
RUN go build -ldflags="-s -w" -o /out/server ./cmd/server && \
    go build -ldflags="-s -w" -o /out/novel  ./cmd/novel

### Stage 2: runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata sqlite-libs && \
    addgroup -S app && adduser -S -G app app

WORKDIR /app
COPY --from=builder /out/server /app/server
COPY --from=builder /out/novel  /app/novel
COPY .env.example /app/.env.example

# data 目录留给 SQLite,挂卷使用
RUN mkdir -p /app/data /app/logs && chown -R app:app /app
USER app

EXPOSE 3000

# 默认 liveness probe(可被 compose / k8s 覆盖)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O - http://127.0.0.1:3000/health || exit 1

ENTRYPOINT ["/app/server"]
