package converter

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	"github.com/you-humble/rocket-maintenance/order/internal/model"
	assemblypbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/assembly/v1"
)

type kafkaConverter struct{}

func NewKafkaCoverter() *kafkaConverter { return &kafkaConverter{} }

func (c *kafkaConverter) PaidOrderToModel(m model.PaidOrder) ([]byte, error) {
	pb := &assemblypbv1.PaidOrderRecord{
		EventUuid:       m.EventID.String(),
		OrderUuid:       m.OrderID.String(),
		UserUuid:        m.UserID.String(),
		PaymentMethod:   string(m.PaymentMethod),
		TransactionUuid: m.TransactionID.String(),
	}

	payload, err := proto.Marshal(pb)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal protobuf: %w", err)
	}

	return payload, nil
}

func (c *kafkaConverter) AssembledShipToModel(data []byte) (model.AssembledShip, error) {
	var pb assemblypbv1.AssembledShipRecord
	if err := proto.Unmarshal(data, &pb); err != nil {
		return model.AssembledShip{}, fmt.Errorf("failed to unmarshal protobuf: %w", err)
	}

	return model.AssembledShip{
		EventID:   uuid.MustParse(pb.GetEventUuid()),
		OrderID:   uuid.MustParse(pb.GetOrderUuid()),
		UserID:    uuid.MustParse(pb.GetUserUuid()),
		BuildTime: time.Duration(pb.BuildTimeSec),
	}, nil
}
