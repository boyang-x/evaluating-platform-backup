FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

ENV GOPROXY=https://goproxy.cn,direct
ENV GONOSUMDB=*

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /out/ccbos-mcp-server .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

WORKDIR /app

COPY --from=builder /out/ccbos-mcp-server /app/ccbos-mcp-server

ENV CCBOS_MCP_PORT=18191 \
    CCBOS_LISTEN_HOST=0.0.0.0 \
    CCBOS_PUBLIC_BASE_URL=http://localhost:18191 \
    CCBOS_API_BASE_URL=https://api.deepseek.com/v1 \
    CCBOS_MODEL=deepseek-chat \
    CCBOS_REQUEST_TIMEOUT_SECONDS=60 \
    CCBOS_ARTIFACT_DIR=/app/artifacts

VOLUME ["/app/artifacts"]

EXPOSE 18191

ENTRYPOINT ["/app/ccbos-mcp-server"]
