package store

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/applycoding/k8s-microservice-deploy/internal/trip"
)

// Memory is an in-process store for tests and STORE=memory.
type Memory struct {
	mu    sync.Mutex
	trips map[string]trip.Trip
}

// NewMemory returns an empty store.
func NewMemory() *Memory {
	return &Memory{trips: map[string]trip.Trip{}}
}

// Ping always succeeds.
func (m *Memory) Ping(context.Context) error {
	return nil
}

// Create stores a trip.
func (m *Memory) Create(_ context.Context, t trip.Trip) (trip.Trip, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.trips[t.ID] = t
	return t, nil
}

// Get returns a trip by id.
func (m *Memory) Get(_ context.Context, id string) (trip.Trip, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.trips[id]
	if !ok {
		return trip.Trip{}, ErrNotFound
	}
	return t, nil
}

// List returns trips newest first.
func (m *Memory) List(_ context.Context, limit int) ([]trip.Trip, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]trip.Trip, 0, len(m.trips))
	for _, t := range m.trips {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// Update changes status and, when driverID is non-nil, the driver.
func (m *Memory) Update(_ context.Context, id string, status trip.Status, driverID *string) (trip.Trip, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.trips[id]
	if !ok {
		return trip.Trip{}, ErrNotFound
	}
	t.Status = status
	if driverID != nil {
		t.DriverID = *driverID
	}
	t.UpdatedAt = time.Now().UTC()
	m.trips[id] = t
	return t, nil
}
