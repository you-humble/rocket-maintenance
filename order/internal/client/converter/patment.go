package converter

import (
	"github.com/you-humble/rocket-maintenance/order/internal/model"
	paymentpbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/payment/v1"
)

func PayOrderParamsToPB(params model.PayOrderParams) *paymentpbv1.PayOrderRequest {
	return &paymentpbv1.PayOrderRequest{
		OrderUuid:     params.ID.String(),
		UserUuid:      params.UserID.String(),
		PaymentMethod: paymentMethodToPB(params.PaymentMethod),
	}
}

func paymentMethodToPB(m model.PaymentMethod) paymentpbv1.PaymentMethod {
	switch m {
	case model.PaymentMethodUnknown:
		return paymentpbv1.PaymentMethod_PAYMENT_METHOD_UNKNOWN
	case model.PaymentMethodCard:
		return paymentpbv1.PaymentMethod_PAYMENT_METHOD_CARD
	case model.PaymentMethodSBP:
		return paymentpbv1.PaymentMethod_PAYMENT_METHOD_SBP
	case model.PaymentMethodCreditCard:
		return paymentpbv1.PaymentMethod_PAYMENT_METHOD_CREDIT_CARD
	case model.PaymentMethodInvestorMoney:
		return paymentpbv1.PaymentMethod_PAYMENT_METHOD_INVESTOR_MONEY
	default:
		return paymentpbv1.PaymentMethod_PAYMENT_METHOD_UNKNOWN
	}
}
