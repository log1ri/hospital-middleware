package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDummyHISSearch(t *testing.T) {
	for _, tt := range []struct {
		id         string
		wantStatus int
	}{
		{"1234567890123", http.StatusOK},
		{"P1234567", http.StatusOK},
		{"unknown", http.StatusNotFound},
	} {
		t.Run(tt.id, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/patient/search/"+tt.id, nil)
			newHandler().ServeHTTP(response, request)
			if response.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d", tt.wantStatus, response.Code)
			}
			if tt.wantStatus == http.StatusOK && !strings.Contains(response.Body.String(), `"patient_hn":"A0001"`) {
				t.Fatalf("unexpected dummy patient: %s", response.Body.String())
			}
		})
	}
}
