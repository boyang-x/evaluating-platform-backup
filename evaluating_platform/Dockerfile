# ── Build Stage ──────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

ENV GOPROXY=https://goproxy.cn,direct
ENV GONOSUMDB=*

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server ./cmd/server

# ── Runtime Stage ─────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates curl tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

WORKDIR /app

COPY --from=builder /app/server .

EXPOSE 8080

ENTRYPOINT ["/app/server"]
