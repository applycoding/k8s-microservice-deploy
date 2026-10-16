package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/applycoding/k8s-microservice-deploy/internal/metrics"
	"github.com/applycoding/k8s-microservice-deploy/internal/store"
	"github.com/applycoding/k8s-microservice-deploy/internal/trip"
)

// Server serves the trips API. The store may be attached after listen so
// liveness stays green while Postgres is still starting.
type Server struct {
	log *slog.Logger

	mu    sync.RWMutex
	store store.Store
}

// New returns a server with no store attached.
func New(log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}
	return &Server{log: log}
}

// SetStore marks the process ready to serve traffic that needs the database.
func (s *Server) SetStore(st store.Store) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store = st
}

func (s *Server) currentStore() (store.Store, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.store == nil {
		return nil, false
	}
	return s.store, true
}

// Handler returns the HTTP routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.instrument("healthz", s.healthz))
	mux.HandleFunc("GET /readyz", s.instrument("readyz", s.readyz))
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.HandleFunc("POST /v1/trips", s.instrument("create_trip", s.createTrip))
	mux.HandleFunc("GET /v1/trips", s.instrument("list_trips", s.listTrips))
	mux.HandleFunc("GET /v1/trips/{id}", s.instrument("get_trip", s.getTrip))
	mux.HandleFunc("PATCH /v1/trips/{id}", s.instrument("update_trip", s.updateTrip))
	return mux
}

func (s *Server) instrument(route string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next(rec, r)
		metrics.Observe(r.Method, route, rec.status, time.Since(start))
		if rec.status >= 500 {
			s.log.Error("request failed", "method", r.Method, "route", route, "status", rec.status)
		}
	}
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	st, ok := s.currentStore()
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "database not ready")
		return
	}
	if err := st.Ping(r.Context()); err != nil {
		s.log.Warn("readiness ping failed", "err", err)
		writeError(w, http.StatusServiceUnavailable, "database not ready")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) createTrip(w http.ResponseWriter, r *http.Request) {
	st, ok := s.currentStore()
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "database not ready")
		return
	}
	var req trip.CreateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	now := time.Now().UTC()
	created, err := st.Create(r.Context(), trip.Trip{
		ID:        uuid.NewString(),
		RiderID:   strings.TrimSpace(req.RiderID),
		DriverID:  strings.TrimSpace(req.DriverID),
		Pickup:    strings.TrimSpace(req.Pickup),
		Dropoff:   strings.TrimSpace(req.Dropoff),
		Status:    trip.StatusRequested,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		s.log.Error("create trip", "err", err)
		writeError(w, http.StatusInternalServerError, "could not create trip")
		return
	}
	metrics.TripsCreated.Inc()
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) listTrips(w http.ResponseWriter, r *http.Request) {
	st, ok := s.currentStore()
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "database not ready")
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 200 {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and 200")
			return
		}
		limit = n
	}
	trips, err := st.List(r.Context(), limit)
	if err != nil {
		s.log.Error("list trips", "err", err)
		writeError(w, http.StatusInternalServerError, "could not list trips")
		return
	}
	if trips == nil {
		trips = []trip.Trip{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"trips": trips})
}

func (s *Server) getTrip(w http.ResponseWriter, r *http.Request) {
	st, ok := s.currentStore()
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "database not ready")
		return
	}
	t, err := st.Get(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "trip not found")
		return
	}
	if err != nil {
		s.log.Error("get trip", "err", err)
		writeError(w, http.StatusInternalServerError, "could not get trip")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) updateTrip(w http.ResponseWriter, r *http.Request) {
	st, ok := s.currentStore()
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "database not ready")
		return
	}
	var req trip.UpdateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	next := trip.Status(req.Status)
	if !next.Valid() {
		writeError(w, http.StatusBadRequest, "status is invalid")
		return
	}
	id := r.PathValue("id")
	current, err := st.Get(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "trip not found")
		return
	}
	if err != nil {
		s.log.Error("get trip", "err", err)
		writeError(w, http.StatusInternalServerError, "could not get trip")
		return
	}
	if !trip.CanTransition(current.Status, next) {
		writeError(w, http.StatusConflict, "status transition is not allowed")
		return
	}
	var driverID *string
	if req.DriverID != nil {
		trimmed := strings.TrimSpace(*req.DriverID)
		if len(trimmed) > 128 {
			writeError(w, http.StatusBadRequest, "field exceeds maximum length")
			return
		}
		driverID = &trimmed
	}
	updated, err := st.Update(r.Context(), id, next, driverID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "trip not found")
		return
	}
	if err != nil {
		s.log.Error("update trip", "err", err)
		writeError(w, http.StatusInternalServerError, "could not update trip")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("unexpected trailing json")
	}
	return nil
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(body); err != nil {
		slog.Error("write json", "err", err)
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}
