package trip

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/murmyauuu/template/api"
)

type TxManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type Service struct {
	repository *Repository
	history    *HistoryRepository
	txManager  TxManager
}

func NewService(repository *Repository, history *HistoryRepository, txManager TxManager) *Service {
	return &Service{repository: repository, history: history, txManager: txManager}
}

func (s *Service) Create(ctx context.Context, data api.TripData) (api.Trip, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return api.Trip{}, fmt.Errorf("generate trip ID: %w", err)
	}
	value := api.Trip{
		Id: id, UserId: data.UserId, DriverId: data.DriverId,
		StartPoint: data.StartPoint, EndPoint: data.EndPoint, Price: data.Price,
		Status: api.Active, StartedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	var created api.Trip
	err = s.txManager.Do(ctx, func(ctx context.Context) error {
		var err error
		created, err = s.repository.Create(ctx, value)
		if err != nil {
			return err
		}
		return s.history.Append(ctx, created.Id, nil, api.Active, created.StartedAt)
	})
	if err != nil {
		return api.Trip{}, err
	}
	return created, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (api.Trip, error) {
	return s.repository.Get(ctx, id)
}

func (s *Service) Finish(ctx context.Context, id uuid.UUID) (api.Trip, error) {
	finishedAt := time.Now().UTC().Truncate(time.Microsecond)
	var finished api.Trip
	err := s.txManager.Do(ctx, func(ctx context.Context) error {
		var err error
		finished, err = s.repository.Finish(ctx, id, finishedAt)
		if err != nil {
			return err
		}
		from := api.Active
		return s.history.Append(ctx, id, &from, api.Completed, finishedAt)
	})
	if err != nil {
		return api.Trip{}, err
	}
	return finished, nil
}
