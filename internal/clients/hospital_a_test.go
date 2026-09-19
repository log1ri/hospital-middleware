package clients_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hospital-middleware/internal/clients"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newClient(fn roundTripFunc) *clients.HospitalAClient {
	return &clients.HospitalAClient{
		BaseURL:    "https://hospital-a.test/",
		HTTPClient: &http.Client{Transport: fn},
	}
}

func respond(status int, body string) roundTripFunc {
	return func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	}
}

func TestSearchReturnsTheResponseBody(t *testing.T) {
	payload := `{"patient_hn":"A0001"}`
	body, err := newClient(respond(http.StatusOK, payload)).Search(context.Background(), "123")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if string(body) != payload {
		t.Errorf("got %s, want %s", body, payload)
	}
}

// The base URL carries a trailing slash and the ID goes into the path, so the
// client has to join them without doubling the separator.
func TestSearchBuildsTheEndpoint(t *testing.T) {
	var got string
	_, _ = newClient(func(r *http.Request) (*http.Response, error) {
		got = r.URL.String()
		return respond(http.StatusOK, `{}`)(r)
	}).Search(context.Background(), "123")

	if want := "https://hospital-a.test/patient/search/123"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// The ID reaches us from a query string, so it must not be able to climb out of
// the search path.
func TestSearchEscapesTheID(t *testing.T) {
	var got string
	_, _ = newClient(func(r *http.Request) (*http.Response, error) {
		got = r.URL.EscapedPath()
		return respond(http.StatusOK, `{}`)(r)
	}).Search(context.Background(), "../../admin secrets")

	if strings.Contains(got, "../") || strings.Contains(got, " ") {
		t.Errorf("the ID was not escaped: %s", got)
	}
	if !strings.HasPrefix(got, "/patient/search/") {
		t.Errorf("the request left the search path: %s", got)
	}
}

func TestSearchMapsNotFoundToASentinel(t *testing.T) {
	_, err := newClient(respond(http.StatusNotFound, "")).Search(context.Background(), "123")
	if !errors.Is(err, clients.ErrPatientNotFound) {
		t.Errorf("got %v, want ErrPatientNotFound", err)
	}
}

func TestSearchReportsOtherStatuses(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusBadGateway, http.StatusUnauthorized} {
		_, err := newClient(respond(status, "")).Search(context.Background(), "123")
		if err == nil {
			t.Errorf("status %d was accepted", status)
		}
		if errors.Is(err, clients.ErrPatientNotFound) {
			t.Errorf("status %d was reported as a missing patient", status)
		}
	}
}

func TestSearchRejectsInvalidJSON(t *testing.T) {
	_, err := newClient(respond(http.StatusOK, `{"patient_hn":`)).Search(context.Background(), "123")
	if err == nil {
		t.Error("a truncated body was accepted")
	}
}

// A broken or hostile HIS must not be able to exhaust our memory.
func TestSearchRejectsAnOversizedBody(t *testing.T) {
	huge := strings.Repeat("a", 2<<20+10)
	_, err := newClient(respond(http.StatusOK, `{"x":"`+huge+`"}`)).Search(context.Background(), "123")
	if err == nil {
		t.Fatal("an oversized body was accepted")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("got %v, want a size error", err)
	}
}

func TestSearchAcceptsABodyAtTheLimit(t *testing.T) {
	filler := strings.Repeat("a", 2<<20-len(`{"x":""}`))
	_, err := newClient(respond(http.StatusOK, `{"x":"`+filler+`"}`)).Search(context.Background(), "123")
	if err != nil {
		t.Errorf("a body within the limit was rejected: %v", err)
	}
}

func TestSearchWrapsTransportFailures(t *testing.T) {
	failure := errors.New("connection refused")
	_, err := newClient(func(*http.Request) (*http.Response, error) {
		return nil, failure
	}).Search(context.Background(), "123")

	if err == nil || !strings.Contains(err.Error(), "Hospital A") {
		t.Errorf("got %v, want an error naming Hospital A", err)
	}
}

// Runs against a real server so Go's own transport, not a stub, decides what a
// cancelled request does.
func TestSearchStopsWhenTheCallerCancels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("the request reached Hospital A after the caller gave up")
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := &clients.HospitalAClient{BaseURL: server.URL, HTTPClient: server.Client()}
	if _, err := client.Search(ctx, "123"); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
}
