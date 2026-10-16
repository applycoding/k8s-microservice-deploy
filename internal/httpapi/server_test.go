package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/applycoding/k8s-microservice-deploy/internal/store"
	"github.com/applycoding/k8s-microservice-deploy/internal/trip"
)

func testServer(t *testing.T, withStore bool) *Server {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := New(log)
	if withStore {
		s.SetStore(store.NewMemory())
	}
	return s
}

func do(s *Server, method, path, body string) *httptest.ResponseRecorder {
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(dst); err != nil {
		t.Fatal(err)
	}
}

func TestHealthz(t *testing.T) {
	rec := do(testServer(t, false), http.MethodGet, "/healthz", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestReadyzWithoutStore(t *testing.T) {
	rec := do(testServer(t, false), http.MethodGet, "/readyz", "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body.String())
	}
}

func TestReadyzWithStore(t *testing.T) {
	rec := do(testServer(t, true), http.MethodGet, "/readyz", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestCreateGetListAndUpdate(t *testing.T) {
	s := testServer(t, true)
	rec := do(s, http.MethodPost, "/v1/trips", `{"rider_id":"rider-1","pickup":"Market","dropoff":"Ferry"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body %s", rec.Code, rec.Body.String())
	}
	var created trip.Trip
	decode(t, rec, &created)
	if created.ID == "" || created.Status != trip.StatusRequested {
		t.Fatalf("created = %+v", created)
	}

	rec = do(s, http.MethodGet, "/v1/trips/"+created.ID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d", rec.Code)
	}

	rec = do(s, http.MethodGet, "/v1/trips", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	var listed struct {
		Trips []trip.Trip `json:"trips"`
	}
	decode(t, rec, &listed)
	if len(listed.Trips) != 1 || listed.Trips[0].ID != created.ID {
		t.Fatalf("list = %+v", listed.Trips)
	}

	rec = do(s, http.MethodPatch, "/v1/trips/"+created.ID, `{"status":"accepted","driver_id":"driver-9"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status = %d body %s", rec.Code, rec.Body.String())
	}
	var updated trip.Trip
	decode(t, rec, &updated)
	if updated.Status != trip.StatusAccepted || updated.DriverID != "driver-9" {
		t.Fatalf("updated = %+v", updated)
	}

	rec = do(s, http.MethodPatch, "/v1/trips/"+created.ID, `{"status":"requested"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("illegal transition status = %d", rec.Code)
	}
}

func TestCreateValidation(t *testing.T) {
	s := testServer(t, true)
	rec := do(s, http.MethodPost, "/v1/trips", `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	rec = do(s, http.MethodPost, "/v1/trips", `not-json`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestNotFound(t *testing.T) {
	rec := do(testServer(t, true), http.MethodGet, "/v1/trips/missing", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestLimitValidation(t *testing.T) {
	rec := do(testServer(t, true), http.MethodGet, "/v1/trips?limit=0", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestMetrics(t *testing.T) {
	s := testServer(t, true)
	rec := do(s, http.MethodPost, "/v1/trips", `{"rider_id":"r","pickup":"a","dropoff":"b"}`)
	if rec.Code != http.StatusCreated {
		t.Fatal(rec.Body.String())
	}
	rec = do(s, http.MethodGet, "/metrics", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("metrics status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "trips_created_total") || !strings.Contains(body, "http_requests_total") {
		t.Fatalf("metrics body missing counters:\n%s", body)
	}
}

func TestUnavailableWhenStoreMissing(t *testing.T) {
	s := testServer(t, false)
	rec := do(s, http.MethodPost, "/v1/trips", `{"rider_id":"r","pickup":"a","dropoff":"b"}`)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestListEmpty(t *testing.T) {
	rec := do(testServer(t, true), http.MethodGet, "/v1/trips", "")
	if rec.Code != http.StatusOK {
		t.Fatal(rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"trips":[]`)) && !bytes.Contains(rec.Body.Bytes(), []byte(`"trips": []`)) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}
