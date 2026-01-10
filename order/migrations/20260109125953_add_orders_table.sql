-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_method') THEN
        CREATE TYPE payment_method AS ENUM (
            'PAYMENT_METHOD_UNKNOWN',
            'PAYMENT_METHOD_CARD',
            'PAYMENT_METHOD_SBP',
            'PAYMENT_METHOD_CREDIT_CARD',
            'PAYMENT_METHOD_INVESTOR_MONEY'
        );
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'order_status') THEN
        CREATE TYPE order_status AS ENUM (
            'PENDING_PAYMENT',
            'PAID',
            'CANCELLED'
        );
    END IF;
END $$;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS orders (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL,
    part_ids uuid[] NOT NULL,
    total_price bigint NOT NULL,
    transaction_id uuid NULL,
    payment_method payment_method NULL,
    status order_status NOT NULL DEFAULT 'PENDING_PAYMENT',

    CONSTRAINT orders_total_price_non_negative CHECK (total_price >= 0),
    CONSTRAINT orders_paid_requires_payment_data CHECK (
        status <> 'PAID' OR (transaction_id IS NOT NULL AND payment_method IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders (user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS orders;
-- +goose StatementEnd
