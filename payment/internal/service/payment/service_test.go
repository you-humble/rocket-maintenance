package service

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/you-humble/rocket-maintenance/payment/internal/model"
)

func TestService_PayOrder(t *testing.T) {
	log.SetOutput(os.Stdout)

	sut := &service{}
	ctx := context.Background()

	tests := []struct {
		name   string
		params model.PayOrderParams

		wantErr     bool
		wantErrIs   error
		wantErrMsg  string
		checkResult func(t *testing.T, res *model.PayOrderResult)
	}{
		{
			name: "ok/returns valid uuid",
			params: model.PayOrderParams{
				OrderID: "order-1",
				UserID:  "user-1",
				Method:  model.MethodCard,
			},
			wantErr: false,
			checkResult: func(t *testing.T, res *model.PayOrderResult) {
				require.NotNil(t, res)
				require.NotEmpty(t, res.TransactionUUID)
				require.NotEqual(t, uuid.Nil.String(), res.TransactionUUID)
				_, parseErr := uuid.Parse(res.TransactionUUID)
				require.NoError(t, parseErr, "transaction uuid must be valid")
			},
		},
		{
			name: "validation/order_id required",
			params: model.PayOrderParams{
				OrderID: "",
				UserID:  "user-1",
				Method:  model.MethodCard,
			},
			wantErr:    true,
			wantErrMsg: "order_id is required",
			checkResult: func(t *testing.T, res *model.PayOrderResult) {
				require.Nil(t, res)
			},
		},
		{
			name: "validation/user_id required",
			params: model.PayOrderParams{
				OrderID: "order-1",
				UserID:  "",
				Method:  model.MethodCard,
			},
			wantErr:    true,
			wantErrMsg: "user_id is required",
			checkResult: func(t *testing.T, res *model.PayOrderResult) {
				require.Nil(t, res)
			},
		},
		{
			name: "validation/method unknown",
			params: model.PayOrderParams{
				OrderID: "order-1",
				UserID:  "user-1",
				Method:  model.MethodUnknown,
			},
			wantErr:    true,
			wantErrMsg: "payment_method is unknown",
			checkResult: func(t *testing.T, res *model.PayOrderResult) {
				require.Nil(t, res)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res, err := sut.PayOrder(ctx, tt.params)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrIs != nil {
					require.ErrorIs(t, err, tt.wantErrIs)
				}
				if tt.wantErrMsg != "" {
					require.EqualError(t, err, tt.wantErrMsg)
				}
			} else {
				require.NoError(t, err)
			}

			if tt.checkResult != nil {
				tt.checkResult(t, res)
			}
		})
	}
}
