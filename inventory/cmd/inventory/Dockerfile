FROM golang:1.25.4 AS builder

WORKDIR /src

COPY inventory/go.mod inventory/go.sum ./inventory/
COPY shared/go.mod shared/go.sum ./shared/

WORKDIR /src/inventory
RUN go mod download

WORKDIR /src
COPY inventory ./inventory
COPY shared ./shared

WORKDIR /src/inventory
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /bin/inventory ./cmd/inventory

FROM alpine:3.21.3 AS runtime

WORKDIR /app

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /bin/inventory /app/inventory

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 50051

ENTRYPOINT [ "/app/inventory" ]
