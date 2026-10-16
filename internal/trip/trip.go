package trip

import (
	"errors"
	"strings"
	"time"
)

// Status is the lifecycle of a trip.
type Status string

const (
	StatusRequested  Status = "requested"
	StatusAccepted   Status = "accepted"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusCancelled  Status = "cancelled"
)

// Valid reports whether s is a known status.
func (s Status) Valid() bool {
	switch s {
	case StatusRequested, StatusAccepted, StatusInProgress, StatusCompleted, StatusCancelled:
		return true
	default:
		return false
	}
}

// CanTransition reports whether a trip may move from one status to another.
// Repeating the current status is allowed so clients can retry a write.
func CanTransition(from, to Status) bool {
	if !from.Valid() || !to.Valid() {
		return false
	}
	if from == to {
		return true
	}
	switch from {
	case StatusRequested:
		return to == StatusAccepted || to == StatusCancelled
	case StatusAccepted:
		return to == StatusInProgress || to == StatusCancelled
	case StatusInProgress:
		return to == StatusCompleted || to == StatusCancelled
	case StatusCompleted, StatusCancelled:
		return false
	default:
		return false
	}
}

// Trip is a single rider journey.
type Trip struct {
	ID        string    `json:"id"`
	RiderID   string    `json:"rider_id"`
	DriverID  string    `json:"driver_id,omitempty"`
	Pickup    string    `json:"pickup"`
	Dropoff   string    `json:"dropoff"`
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateRequest is the body of POST /v1/trips.
type CreateRequest struct {
	RiderID  string `json:"rider_id"`
	DriverID string `json:"driver_id,omitempty"`
	Pickup   string `json:"pickup"`
	Dropoff  string `json:"dropoff"`
}

// Validate checks required fields and length caps.
func (r CreateRequest) Validate() error {
	switch {
	case strings.TrimSpace(r.RiderID) == "":
		return errors.New("rider_id is required")
	case strings.TrimSpace(r.Pickup) == "":
		return errors.New("pickup is required")
	case strings.TrimSpace(r.Dropoff) == "":
		return errors.New("dropoff is required")
	case len(r.RiderID) > 128 || len(r.DriverID) > 128 || len(r.Pickup) > 200 || len(r.Dropoff) > 200:
		return errors.New("field exceeds maximum length")
	default:
		return nil
	}
}

// UpdateRequest is the body of PATCH /v1/trips/{id}.
// A nil DriverID leaves the stored driver unchanged.
type UpdateRequest struct {
	Status   string  `json:"status"`
	DriverID *string `json:"driver_id"`
}
