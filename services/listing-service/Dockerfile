FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache wget

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main ./cmd/

# grpc_health_probe для healthcheck в docker-compose
ARG GRPC_HEALTH_PROBE_VERSION=v0.4.24
RUN wget -qO/bin/grpc_health_probe \
      https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 \
    && chmod +x /bin/grpc_health_probe

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/config ./config
COPY --from=builder /bin/grpc_health_probe /bin/grpc_health_probe

CMD ["./main"]