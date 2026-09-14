# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app

ENV CGO_ENABLED=0
ENV GOPROXY=https://goproxy.cn,direct

# Cache dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy source including embedded frontend dist
COPY . .

# Build the single static binary (frontend is embedded via go:embed)
RUN go build -o /out/server ./cmd/server

# Runtime stage
FROM alpine:3.20
RUN adduser -D -H appuser
WORKDIR /app

# Runtime deps for TLS / DNS (gin + postgres + https)
RUN apk add --no-cache ca-certificates tzdata wget

COPY --from=builder /out/server /app/server
RUN chmod +x /app/server

USER appuser

EXPOSE 8080

# App reads DB_DRIVER, DB_DSN, APP_ENV, SESSION_KEY, GAIOS_AGENT_API_KEY, PORT from env
ENV APP_ENV=production \
    DB_DRIVER=postgres \
    PORT=8080

CMD ["/app/server"]