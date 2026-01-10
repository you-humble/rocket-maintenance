package repository

import (
	"context"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/you-humble/rocket-maintenance/order/internal/model"
)

type repository struct {
	pool *pgxpool.Pool
	sb   sq.StatementBuilderType
}

func NewOrderRepository(pool *pgxpool.Pool) *repository {
	return &repository{
		pool: pool,
		sb:   sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (r *repository) Create(ctx context.Context, ord *model.Order) (uuid.UUID, error) {
	q := r.sb.
		Insert("orders").
		Columns("user_id", "part_ids", "total_price", "transaction_id", "payment_method", "status").
		Values(ord.UserID, ord.PartIDs, ord.TotalPrice, ord.TransactionID, ord.PaymentMethod, ord.Status).
		Suffix("RETURNING id")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return uuid.Nil, err
	}

	var orderID uuid.UUID
	if err := r.pool.QueryRow(ctx, sqlStr, args...).Scan(&orderID); err != nil {
		return uuid.Nil, err
	}

	return orderID, nil
}

func (r *repository) OrderByID(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	q := r.sb.
		Select("id", "user_id", "part_ids", "total_price", "transaction_id", "payment_method", "status").
		From("orders").
		Where(sq.Eq{"id": id})

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}

	var ord model.Order
	err = r.pool.QueryRow(ctx, sqlStr, args...).Scan(
		&ord.ID,
		&ord.UserID,
		&ord.PartIDs,
		&ord.TotalPrice,
		&ord.TransactionID,
		&ord.PaymentMethod,
		&ord.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrOrderNotFound
		}
		return nil, err
	}

	return &ord, nil
}

func (r *repository) Update(ctx context.Context, upd *model.Order) error {
	if upd.ID == uuid.Nil {
		return errors.New("empty order id")
	}

	if upd.Status == model.StatusPaid {
		if upd.TransactionID == nil || upd.PaymentMethod == nil {
			return errors.New("setting status=PAID requires transaction_id and payment_method")
		}
	}

	set := sq.Eq{}

	if len(upd.PartIDs) > 0 {
		set["part_ids"] = upd.PartIDs
	}
	if upd.TotalPrice != 0 {
		set["total_price"] = upd.TotalPrice
	}
	if upd.TransactionID != nil {
		set["transaction_id"] = upd.TransactionID
	}
	if upd.PaymentMethod != nil {
		set["payment_method"] = upd.PaymentMethod
	}
	if upd.Status != "" {
		set["status"] = upd.Status
	}

	if len(set) == 0 {
		return nil
	}

	q := r.sb.
		Update("orders").
		SetMap(set).
		Where(sq.Eq{"id": upd.ID})

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return err
	}

	ct, err := r.pool.Exec(ctx, sqlStr, args...)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return model.ErrOrderNotFound
	}

	return nil
}
