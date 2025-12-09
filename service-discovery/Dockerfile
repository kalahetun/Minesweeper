# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# 安装必要的构建工具
RUN apk add --no-cache git ca-certificates

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建二进制文件
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /service-discovery \
    ./cmd/main.go

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# 安装 CA 证书（用于 HTTPS 连接）
RUN apk add --no-cache ca-certificates tzdata

# 从构建阶段复制二进制文件
COPY --from=builder /service-discovery /app/service-discovery

# 复制默认配置文件
COPY config.example.yaml /app/config.yaml

# 创建非 root 用户
RUN adduser -D -g '' appuser
USER appuser

# 健康检查（简单检查进程是否存在）
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD pgrep service-discovery || exit 1

# 入口点
ENTRYPOINT ["/app/service-discovery"]

# 默认参数
CMD ["--config", "/app/config.yaml"]
