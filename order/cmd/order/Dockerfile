FROM golang:1.25.4 AS builder

WORKDIR /src

COPY order/go.mod order/go.sum ./order/
COPY shared/go.mod shared/go.sum ./shared/

WORKDIR /src/order
RUN go mod download

WORKDIR /src
COPY order ./order
COPY shared ./shared

WORKDIR /src/order
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /bin/order ./cmd/order

FROM alpine:3.21.3 AS runtime

WORKDIR /app

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /bin/order /app/order

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

ENTRYPOINT [ "/app/order" ]
