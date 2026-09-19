package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"hospital-middleware/internal/handlers"
)

func TestHealthReportsOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/health", handlers.Health)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))

	if response.Code != http.StatusOK {
		t.Errorf("got %d, want 200", response.Code)
	}
	if got := strings.TrimSpace(response.Body.String()); got != `{"status":"ok"}` {
		t.Errorf("got %s", got)
	}
}
