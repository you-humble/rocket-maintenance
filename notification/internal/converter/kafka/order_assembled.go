package converter

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	"github.com/you-humble/rocket-maintenance/notification/internal/model"
	assemblypbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/assembly/v1"
)

type kafkaConverter struct{}

func NewKafkaCoverter() *kafkaConverter { return &kafkaConverter{} }

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

func (c *kafkaConverter) PaidOrderToModel(data []byte) (model.PaidOrder, error) {
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
