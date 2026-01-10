package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/you-humble/rocket-maintenance/inventory/internal/model"
	"github.com/you-humble/rocket-maintenance/inventory/internal/service/mocks"
)

func TestServicePart(t *testing.T) {
	t.Parallel()

	type deps struct {
		repository *mocks.MockPartRepository
	}

	newSvc := func(d deps) *service {
		return NewInventoryService(d.repository, 5*time.Second)
	}

	type testCase struct {
		name   string
		partID string
		setup  func(d deps)
		assert func(t *testing.T, res *model.Part, err error, d deps)
	}

	partID := gofakeit.UUID()
	wantPart := &model.Part{
		ID:       partID,
		Name:     gofakeit.ProductName(),
		Category: model.CategoryEngine,
		Tags:     []string{gofakeit.Word(), gofakeit.Word()},
		Manufacturer: &model.Manufacturer{
			Country: gofakeit.Country(),
		},
	}

	tests := []testCase{
		{
			name:   "validation error: empty uuid after trim",
			partID: "   ",
			setup: func(d deps) {
				// No calls expected.
			},
			assert: func(t *testing.T, res *model.Part, err error, d deps) {
				require.Error(t, err)
				assert.ErrorIs(t, err, model.ErrInvalidArgument)
				assert.ErrorContains(t, err, "uuid must be non-empty")
				assert.Nil(t, res)

				d.repository.AssertNotCalled(t, "PartByID", mock.Anything, mock.Anything)
				d.repository.AssertExpectations(t)
			},
		},
		{
			name:   "repository error: PartByID returns error",
			partID: "  " + partID + "  ",
			setup: func(d deps) {
				// Ensure service passes trimmed uuid.
				d.repository.
					On("PartByID", mock.Anything, partID).
					Return((*model.Part)(nil), errors.New("db read failed")).
					Once()
			},
			assert: func(t *testing.T, res *model.Part, err error, d deps) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "db read failed")
				assert.Nil(t, res)

				d.repository.AssertExpectations(t)
			},
		},
		{
			name:   "success: trims uuid and returns part",
			partID: " \n\t" + partID + "\t ",
			setup: func(d deps) {
				d.repository.
					On("PartByID", mock.Anything, partID).
					Return(wantPart, nil).
					Once()
			},
			assert: func(t *testing.T, res *model.Part, err error, d deps) {
				require.NoError(t, err)
				require.NotNil(t, res)
				assert.Equal(t, wantPart, res)

				d.repository.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := deps{
				repository: mocks.NewMockPartRepository(t),
			}
			if tt.setup != nil {
				tt.setup(d)
			}

			svc := newSvc(d)

			res, err := svc.Part(context.Background(), tt.partID)
			tt.assert(t, res, err, d)
		})
	}
}

func TestServiceListParts(t *testing.T) {
	t.Parallel()

	type deps struct {
		repository *mocks.MockPartRepository
	}

	newSvc := func(d deps) *service {
		return NewInventoryService(d.repository, 5*time.Second)
	}

	// Stable test data set.
	p1 := &model.Part{
		ID:       "id-1",
		Name:     "Bolt",
		Category: model.CategoryEngine,
	}
	p2 := &model.Part{
		ID:       "id-2",
		Name:     "Nut",
		Category: model.CategoryFuel,
	}
	p3 := &model.Part{
		ID:       "id-3",
		Name:     "Washer",
		Category: model.CategoryFuel,
	}
	p4 := &model.Part{
		ID:       "id-4",
		Name:     "Motor",
		Category: model.CategoryPorthole,
	}

	all := []*model.Part{p1, p2, p3, p4}

	type testCase struct {
		name   string
		filter model.PartsFilter
		setup  func(d deps)
		assert func(t *testing.T, res []*model.Part, err error, d deps)
	}

	tests := []testCase{
		{
			name:   "repository error: List returns error",
			filter: model.PartsFilter{},
			setup: func(d deps) {
				d.repository.
					On("List", mock.Anything, model.PartsFilter{}).
					Return(([]*model.Part)(nil), errors.New("db read failed")).
					Once()
			},
			assert: func(t *testing.T, res []*model.Part, err error, d deps) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "db read failed")
				assert.Nil(t, res)
				d.repository.AssertExpectations(t)
			},
		},
		{
			name:   "empty filter: service returns repository result as-is",
			filter: model.PartsFilter{},
			setup: func(d deps) {
				d.repository.
					On("List", mock.Anything, model.PartsFilter{}).
					Return(all, nil).
					Once()
			},
			assert: func(t *testing.T, res []*model.Part, err error, d deps) {
				require.NoError(t, err)
				assert.Equal(t, all, res)
				d.repository.AssertExpectations(t)
			},
		},
		{
			name: "non-empty filter: service passes filter to repository and returns its result",
			filter: model.PartsFilter{
				IDs: []string{"id-2", "id-3"},
			},
			setup: func(d deps) {
				d.repository.
					On("List", mock.Anything, model.PartsFilter{IDs: []string{"id-2", "id-3"}}).
					Return([]*model.Part{p2, p3}, nil).
					Once()
			},
			assert: func(t *testing.T, res []*model.Part, err error, d deps) {
				require.NoError(t, err)
				assert.Equal(t, []*model.Part{p2, p3}, res)
				d.repository.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := deps{
				repository: mocks.NewMockPartRepository(t),
			}
			if tt.setup != nil {
				tt.setup(d)
			}

			svc := newSvc(d)

			res, err := svc.ListParts(context.Background(), tt.filter)
			tt.assert(t, res, err, d)
		})
	}
}
