FROM golang:1.25.4 AS builder

WORKDIR /src

COPY inventory/go.mod inventory/go.sum ./inventory/
COPY shared/go.mod shared/go.sum ./shared/
COPY platform/go.mod platform/go.sum ./platform/

WORKDIR /src/inventory
RUN go mod download

WORKDIR /src
COPY inventory ./inventory
COPY shared ./shared
COPY platform ./platform

WORKDIR /src/inventory
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /bin/inventory ./cmd/inventory

FROM alpine:3.21.3 AS runtime

WORKDIR /app

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

ARG GRPCURL_VERSION=1.9.3
RUN apk add --no-cache ca-certificates curl && update-ca-certificates && \
    curl -fsSL -o /tmp/grpcurl.tar.gz \
    "https://github.com/fullstorydev/grpcurl/releases/download/v${GRPCURL_VERSION}/grpcurl_${GRPCURL_VERSION}_linux_x86_64.tar.gz" && \
    tar -xzf /tmp/grpcurl.tar.gz -C /usr/local/bin grpcurl && \
    chmod +x /usr/local/bin/grpcurl && \
    rm -f /tmp/grpcurl.tar.gz

COPY --from=builder /bin/inventory /app/inventory

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 50051

ENTRYPOINT [ "/app/inventory" ]
