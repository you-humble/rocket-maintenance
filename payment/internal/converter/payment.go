package converter

import (
	"errors"

	"github.com/you-humble/rocket-maintenance/payment/internal/model"
	paymentpbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/payment/v1"
)

func methodFromPB(m paymentpbv1.PaymentMethod) (model.Method, error) {
	switch m {
	case paymentpbv1.PaymentMethod_PAYMENT_METHOD_UNKNOWN:
		return model.MethodUnknown, errors.New("payment_method unknown")
	case paymentpbv1.PaymentMethod_PAYMENT_METHOD_CARD:
		return model.MethodCard, nil
	case paymentpbv1.PaymentMethod_PAYMENT_METHOD_SBP:
		return model.MethodSBP, nil
	case paymentpbv1.PaymentMethod_PAYMENT_METHOD_CREDIT_CARD:
		return model.MethodCreditCard, nil
	case paymentpbv1.PaymentMethod_PAYMENT_METHOD_INVESTOR_MONEY:
		return model.MethodInvestorMoney, nil
	default:
		return model.MethodUnknown, errors.New("payment_method unsupported")
	}
}

func PayOrderParamsFromPB(req *paymentpbv1.PayOrderRequest) (model.PayOrderParams, error) {
	if req == nil {
		return model.PayOrderParams{}, errors.New("request is nil")
	}
	m, err := methodFromPB(req.GetPaymentMethod())
	if err != nil {
		return model.PayOrderParams{}, err
	}
	return model.PayOrderParams{
		OrderID: req.GetOrderUuid(),
		UserID:  req.GetUserUuid(),
		Method:  m,
	}, nil
}

func PayOrderRespToPB(res *model.PayOrderResult) *paymentpbv1.PayOrderResponse {
	return &paymentpbv1.PayOrderResponse{
		TransactionUuid: res.TransactionUUID,
	}
}
