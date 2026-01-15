package converter

import (
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	"github.com/you-humble/rocket-maintenance/assembly/internal/model"
	assemblypbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/assembly/v1"
)

type converter struct{}

func NewKafkaCoverter() *converter { return &converter{} }

func (c *converter) PaidOrderToModel(data []byte) (model.PaidOrder, error) {
	var pb assemblypbv1.PaidOrderRecord
	if err := proto.Unmarshal(data, &pb); err != nil {
		return model.PaidOrder{}, fmt.Errorf("failed to unmarshal protobuf: %w", err)
	}

	return model.PaidOrder{
		EventID:       uuid.MustParse(pb.GetEventUuid()),
		OrderID:       uuid.MustParse(pb.GetOrderUuid()),
		UserID:        uuid.MustParse(pb.GetUserUuid()),
		PaymentMethod: pb.GetPaymentMethod(),
		TransactionID: uuid.MustParse(pb.GetTransactionUuid()),
	}, nil
}

func (c *converter) AssembledShipToPayload(m model.AssembledShip) ([]byte, error) {
	pb := &assemblypbv1.AssembledShipRecord{
		EventUuid:    m.EventID.String(),
		OrderUuid:    m.OrderID.String(),
		UserUuid:     m.UserID.String(),
		BuildTimeSec: m.BuildTime.Milliseconds() / 1000,
	}

	payload, err := proto.Marshal(pb)
	if err != nil {
		return nil, fmt.Errorf("failed to mashal protobuf: %w", err)
	}

	return payload, nil
}
