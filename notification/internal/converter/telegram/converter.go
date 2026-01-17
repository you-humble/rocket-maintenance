package converter

import (
	"bytes"
	"embed"
	"text/template"

	"github.com/you-humble/rocket-maintenance/notification/internal/model"
)

var (
	//go:embed templates/order_paid.tmpl
	orderPaidFS       embed.FS
	orderPaidTemplate = template.Must(template.ParseFS(orderPaidFS, "templates/order_paid.tmpl"))

	//go:embed templates/ship_assembled.tmpl
	shipAssembledFS       embed.FS
	shipAssembledTemplate = template.Must(template.ParseFS(shipAssembledFS, "templates/ship_assembled.tmpl"))
)

func BuildPaidOrder(event model.PaidOrder) (string, error) {
	n := model.PaidOrderNotification{
		OrderID:       event.OrderID.String(),
		UserID:        event.UserID.String(),
		PaymentMethod: event.PaymentMethod,
		TransactionID: event.TransactionID.String(),
	}

	var buf bytes.Buffer
	if err := orderPaidTemplate.Execute(&buf, n); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func BuildShipAssembled(event model.AssembledShip) (string, error) {
	n := model.AssembledShipNotification{
		OrderID:   event.OrderID.String(),
		UserID:    event.UserID.String(),
		BuildTime: event.BuildTime,
	}

	var buf bytes.Buffer
	if err := shipAssembledTemplate.Execute(&buf, n); err != nil {
		return "", err
	}

	return buf.String(), nil
}
