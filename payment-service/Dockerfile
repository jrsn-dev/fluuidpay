# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/payment-service ./cmd/api

# Runtime stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/payment-service .
COPY --from=builder /build/config ./config
COPY --from=builder /build/migrations ./migrations
COPY --from=builder /build/openapi-payment-ibs-cbs.yaml .

EXPOSE 8080

ENTRYPOINT ["./payment-service"]
