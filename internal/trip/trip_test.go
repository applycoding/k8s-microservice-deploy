package trip

import "testing"

func TestStatusValid(t *testing.T) {
	if !StatusRequested.Valid() {
		t.Fatal("requested should be valid")
	}
	if Status("nope").Valid() {
		t.Fatal("unknown status should be invalid")
	}
}

func TestCanTransition(t *testing.T) {
	cases := []struct {
		from Status
		to   Status
		ok   bool
	}{
		{StatusRequested, StatusRequested, true},
		{StatusRequested, StatusAccepted, true},
		{StatusRequested, StatusCancelled, true},
		{StatusRequested, StatusInProgress, false},
		{StatusRequested, StatusCompleted, false},
		{StatusAccepted, StatusInProgress, true},
		{StatusAccepted, StatusCancelled, true},
		{StatusAccepted, StatusCompleted, false},
		{StatusInProgress, StatusCompleted, true},
		{StatusInProgress, StatusCancelled, true},
		{StatusInProgress, StatusAccepted, false},
		{StatusCompleted, StatusCancelled, false},
		{StatusCancelled, StatusRequested, false},
		{Status("nope"), StatusAccepted, false},
	}
	for _, tc := range cases {
		if got := CanTransition(tc.from, tc.to); got != tc.ok {
			t.Errorf("CanTransition(%s, %s) = %v, want %v", tc.from, tc.to, got, tc.ok)
		}
	}
}

func TestCreateRequestValidate(t *testing.T) {
	ok := CreateRequest{RiderID: "r", Pickup: "a", Dropoff: "b"}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (CreateRequest{Pickup: "a", Dropoff: "b"}).Validate(); err == nil {
		t.Fatal("expected rider_id error")
	}
	if err := (CreateRequest{RiderID: "r", Dropoff: "b"}).Validate(); err == nil {
		t.Fatal("expected pickup error")
	}
	long := make([]byte, 201)
	for i := range long {
		long[i] = 'x'
	}
	if err := (CreateRequest{RiderID: "r", Pickup: string(long), Dropoff: "b"}).Validate(); err == nil {
		t.Fatal("expected length error")
	}
}
