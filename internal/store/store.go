package store

import (
	"context"
	"errors"

	"github.com/applycoding/k8s-microservice-deploy/internal/trip"
)

// ErrNotFound is returned when a trip id does not exist.
var ErrNotFound = errors.New("trip not found")

// Store is the persistence used by the HTTP handlers.
type Store interface {
	Ping(ctx context.Context) error
	Create(ctx context.Context, t trip.Trip) (trip.Trip, error)
	Get(ctx context.Context, id string) (trip.Trip, error)
	List(ctx context.Context, limit int) ([]trip.Trip, error)
	Update(ctx context.Context, id string, status trip.Status, driverID *string) (trip.Trip, error)
}
