package converter

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/you-humble/rocket-maintenance/order/internal/model"
	orderv1 "github.com/you-humble/rocket-maintenance/shared/pkg/openapi/order/v1"
	inventorypbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/inventory/v1"
)

func OAPIToPaymentMethod(pm orderv1.PaymentMethod) model.PaymentMethod {
	switch pm {
	case orderv1.PaymentMethodPAYMENTMETHODUNKNOWN:
		return model.PaymentMethodUnknown
	case orderv1.PaymentMethodPAYMENTMETHODCARD:
		return model.PaymentMethodCard
	case orderv1.PaymentMethodPAYMENTMETHODSBP:
		return model.PaymentMethodSBP
	case orderv1.PaymentMethodPAYMENTMETHODCREDITCARD:
		return model.PaymentMethodCreditCard
	case orderv1.PaymentMethodPAYMENTMETHODINVESTORMONEY:
		return model.PaymentMethodInvestorMoney
	default:
		return model.PaymentMethodUnknown
	}
}

func CategoryToPB(c model.Category) inventorypbv1.Category {
	switch c {
	case model.CategoryUnknown:
		return inventorypbv1.Category_CATEGORY_UNKNOWN
	case model.CategoryEngine:
		return inventorypbv1.Category_CATEGORY_ENGINE
	case model.CategoryFuel:
		return inventorypbv1.Category_CATEGORY_FUEL
	case model.CategoryPorthole:
		return inventorypbv1.Category_CATEGORY_PORTHOLE
	case model.CategoryWing:
		return inventorypbv1.Category_CATEGORY_WING
	default:
		return inventorypbv1.Category_CATEGORY_UNKNOWN
	}
}

func CreateOrderRequestToParams(req *orderv1.CreateOrderRequest) model.CreateOrderParams {
	return model.CreateOrderParams{
		UserID:  req.UserUUID,
		PartIDs: req.PartUuids,
	}
}

func CreateOrderResultToResponse(res *model.CreateOrderResult) orderv1.CreateOrderRes {
	return &orderv1.CreateOrderResponse{
		UUID:       res.ID,
		TotalPrice: formatCents(res.TotalPrice),
	}
}

func PayOrderRequestToParams(ordID uuid.UUID, req *orderv1.PayOrderRequest) model.PayOrderParams {
	return model.PayOrderParams{
		ID:            ordID,
		PaymentMethod: OAPIToPaymentMethod(req.PaymentMethod),
	}
}

func PayOrderResultToResponse(res *model.PayOrderResult) orderv1.PayOrderRes {
	return &orderv1.PayOrderResponse{
		TransactionUUID: res.TransactionID,
	}
}

func OrderToOAPI(m *model.Order) *orderv1.Order {
	if m == nil {
		return nil
	}

	return &orderv1.Order{
		OrderUUID:       m.ID,
		UserUUID:        m.UserID,
		PartUuids:       append([]uuid.UUID(nil), m.PartIDs...),
		TotalPrice:      formatCents(m.TotalPrice),
		TransactionUUID: transactionIDToOptNilUUID(m.TransactionID),
		PaymentMethod:   paymentMethodToOptNil(m.PaymentMethod),
		Status:          orderStatusToOAPI(m.Status),
	}
}

func transactionIDToOptNilUUID(id *uuid.UUID) orderv1.OptNilUUID {
	if id == nil {
		return orderv1.OptNilUUID{
			Set:  true,
			Null: true,
		}
	}

	return orderv1.OptNilUUID{
		Value: *id,
		Set:   true,
		Null:  false,
	}
}

func paymentMethodToOptNil(pm *model.PaymentMethod) orderv1.OptNilPaymentMethod {
	if pm == nil {
		return orderv1.OptNilPaymentMethod{
			Set:  true,
			Null: true,
		}
	}

	return orderv1.OptNilPaymentMethod{
		Value: orderv1.PaymentMethod(*pm),
		Set:   true,
		Null:  false,
	}
}

func orderStatusToOAPI(s model.OrderStatus) orderv1.OrderStatus {
	switch s {
	case model.StatusPendingPayment:
		return orderv1.OrderStatusPENDINGPAYMENT
	case model.StatusPaid:
		return orderv1.OrderStatusPAID
	case model.StatusCancelled:
		return orderv1.OrderStatusCANCELLED
	default:
		return orderv1.OrderStatusPENDINGPAYMENT
	}
}

func formatCents(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}
