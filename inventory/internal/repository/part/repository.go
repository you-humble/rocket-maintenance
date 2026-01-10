package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/you-humble/rocket-maintenance/inventory/internal/model"
)

type repository struct {
	coll *mongo.Collection
}

func NewPartRepository(collection *mongo.Collection) *repository {
	return &repository{coll: collection}
}

func (s *repository) PartByID(ctx context.Context, id string) (*model.Part, error) {
	const op = "repository.PartByID"

	var ent PartEntity
	err := s.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&ent)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, model.ErrPartNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return EntityToModel(&ent), nil
}

func (r *repository) List(ctx context.Context, filter model.PartsFilter) ([]*model.Part, error) {
	const op = "repository.List"

	cur, err := r.coll.Find(ctx, BuildMongoFilter(filter))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if cerr := cur.Close(ctx); err != nil {
			log.Printf("%s failed to close cursor: %s", op, cerr)
			return
		}
	}()

	out := make([]*model.Part, 0)
	for cur.Next(ctx) {
		var ent PartEntity
		if err := cur.Decode(&ent); err != nil {
			return nil, fmt.Errorf("%s decode: %w", op, err)
		}
		out = append(out, EntityToModel(&ent))
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("%s cursor: %w", op, err)
	}

	return out, nil
}

func (r *repository) CreateBatch(ctx context.Context, parts []*model.Part) error {
	const op = "repository.CreateBatch"

	docs := make([]any, 0, len(parts))
	for _, p := range parts {
		if p == nil {
			continue
		}
		if p.ID == "" {
			return fmt.Errorf("%s: part ID is empty", op)
		}
		if p.CreatedAt == nil || p.CreatedAt.IsZero() {
			p.CreatedAt = lo.ToPtr(time.Now())
		}

		docs = append(docs, EntityFromModel(p))
	}
	if len(docs) == 0 {
		return nil
	}

	_, err := r.coll.InsertMany(ctx, docs, options.InsertMany().SetOrdered(false))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
