package model

import "errors"

type Method int

const (
	MethodUnknown Method = iota
	MethodCard
	MethodSBP
	MethodCreditCard
	MethodInvestorMoney
)

type PayOrderParams struct {
	OrderID string
	UserID  string
	Method  Method
}

func (p PayOrderParams) Validate() error {
	if p.OrderID == "" {
		return errors.New("order_id is required")
	}
	if p.UserID == "" {
		return errors.New("user_id is required")
	}
	if p.Method == MethodUnknown {
		return errors.New("payment_method is unknown")
	}
	return nil
}

type PayOrderResult struct {
	TransactionUUID string
}
