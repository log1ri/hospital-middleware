package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hospital-middleware/internal/models"
	"hospital-middleware/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type existingStaffStore struct{}

func (existingStaffStore) GetStaffByUsername(string) (*models.Staff, error) {
	return &models.Staff{}, nil
}

type loginErrorStore struct {
	err error
}

func (s loginErrorStore) GetStaffByUsername(string) (*models.Staff, error) {
	return nil, s.err
}

func (loginErrorStore) CreateStaff(*models.Staff) error {
	panic("CreateStaff must not be called during login")
}

func TestLoginStaffMapsErrorsToHTTPStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name       string
		lookupErr  error
		wantStatus int
		wantError  string
	}{
		{"unknown user", gorm.ErrRecordNotFound, http.StatusUnauthorized, "invalid credentials"},
		{"database failure", errors.New("database unavailable"), http.StatusInternalServerError, "could not log in"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			handler := StaffHandler{Service: &services.StaffService{Repo: loginErrorStore{err: tc.lookupErr}}}
			router.POST("/staff/login", handler.LoginStaff)

			request := httptest.NewRequest(http.MethodPost, "/staff/login", strings.NewReader(`{"username":"alice","password":"password","hospital":"Hospital A"}`))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d: %s", tc.wantStatus, response.Code, response.Body.String())
			}
			var body struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Error != tc.wantError {
				t.Fatalf("expected error %q, got %q", tc.wantError, body.Error)
			}
		})
	}
}

func (existingStaffStore) CreateStaff(*models.Staff) error {
	panic("CreateStaff must not be called for an existing username")
}

func TestCreateStaffReturnsConflictForExistingUsername(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := StaffHandler{Service: &services.StaffService{Repo: existingStaffStore{}}}
	router.POST("/staff/create", handler.CreateStaff)

	request := httptest.NewRequest(http.MethodPost, "/staff/create", strings.NewReader(`{"username":"alice","password":"password","hospital":"Hospital A"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", response.Code, response.Body.String())
	}
}
