# Stage 1: Build the Go binary
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN GOPROXY=https://goproxy.cn,direct go mod download

# Build
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o /server ./cmd/server

# Stage 2: Minimal runtime
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# Set timezone
ENV TZ=Asia/Shanghai

COPY --from=builder /server /server

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/server"]
