package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/applycoding/k8s-microservice-deploy/internal/trip"
)

const createTable = `
CREATE TABLE IF NOT EXISTS trips (
    id TEXT PRIMARY KEY,
    rider_id TEXT NOT NULL,
    driver_id TEXT NOT NULL DEFAULT '',
    pickup TEXT NOT NULL,
    dropoff TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
)`

const createIndex = `
CREATE INDEX IF NOT EXISTS trips_created_at_idx ON trips (created_at DESC)`

// Postgres stores trips in PostgreSQL. The schema is created on connect.
// This demo applies DDL at startup instead of running a migration tool.
type Postgres struct {
	pool *pgxpool.Pool
}

// Connect opens a pool and applies the demo schema.
func Connect(ctx context.Context, dsn string) (*Postgres, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 5
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	p := &Postgres{pool: pool}
	if err := p.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return p, nil
}

// Close releases the pool.
func (p *Postgres) Close() {
	p.pool.Close()
}

func (p *Postgres) migrate(ctx context.Context) error {
	for _, stmt := range []string{createTable, createIndex} {
		if _, err := p.pool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

// Ping checks the database.
func (p *Postgres) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// Create inserts a trip.
func (p *Postgres) Create(ctx context.Context, t trip.Trip) (trip.Trip, error) {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO trips (id, rider_id, driver_id, pickup, dropoff, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, t.ID, t.RiderID, t.DriverID, t.Pickup, t.Dropoff, string(t.Status), t.CreatedAt, t.UpdatedAt)
	if err != nil {
		return trip.Trip{}, err
	}
	return t, nil
}

// Get loads one trip.
func (p *Postgres) Get(ctx context.Context, id string) (trip.Trip, error) {
	row := p.pool.QueryRow(ctx, `
		SELECT id, rider_id, driver_id, pickup, dropoff, status, created_at, updated_at
		FROM trips
		WHERE id = $1
	`, id)
	return scanTrip(row)
}

// List returns trips newest first.
func (p *Postgres) List(ctx context.Context, limit int) ([]trip.Trip, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, rider_id, driver_id, pickup, dropoff, status, created_at, updated_at
		FROM trips
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []trip.Trip{}
	for rows.Next() {
		t, err := scanTrip(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// Update changes status and, when driverID is non-nil, the driver.
func (p *Postgres) Update(ctx context.Context, id string, status trip.Status, driverID *string) (trip.Trip, error) {
	now := time.Now().UTC()
	var row pgx.Row
	if driverID == nil {
		row = p.pool.QueryRow(ctx, `
			UPDATE trips
			SET status = $2, updated_at = $3
			WHERE id = $1
			RETURNING id, rider_id, driver_id, pickup, dropoff, status, created_at, updated_at
		`, id, string(status), now)
	} else {
		row = p.pool.QueryRow(ctx, `
			UPDATE trips
			SET status = $2, driver_id = $3, updated_at = $4
			WHERE id = $1
			RETURNING id, rider_id, driver_id, pickup, dropoff, status, created_at, updated_at
		`, id, string(status), *driverID, now)
	}
	return scanTrip(row)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanTrip(row scanner) (trip.Trip, error) {
	var (
		t      trip.Trip
		status string
	)
	err := row.Scan(&t.ID, &t.RiderID, &t.DriverID, &t.Pickup, &t.Dropoff, &status, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return trip.Trip{}, ErrNotFound
	}
	if err != nil {
		return trip.Trip{}, err
	}
	t.Status = trip.Status(status)
	return t, nil
}
