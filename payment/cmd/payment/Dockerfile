FROM golang:1.25.4 AS builder

WORKDIR /src

COPY payment/go.mod payment/go.sum ./payment/
COPY shared/go.mod shared/go.sum ./shared/

WORKDIR /src/payment
RUN go mod download

WORKDIR /src
COPY payment ./payment
COPY shared ./shared

WORKDIR /src/payment
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /bin/payment ./cmd/payment

FROM alpine:3.21.3 AS runtime

WORKDIR /app

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /bin/payment /app/payment

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 50052

ENTRYPOINT [ "/app/payment" ]
