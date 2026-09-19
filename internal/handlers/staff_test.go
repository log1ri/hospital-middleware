package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"hospital-middleware/internal/handlers"
	"hospital-middleware/internal/models"
	"hospital-middleware/internal/services"
	"hospital-middleware/internal/utils"
)

// staffStore is the smallest store that lets create and login run for real
// against the actual hashing and token code. lookupErr injects a failure the
// in-memory map cannot produce on its own, and createCalls lets a test assert
// that a path never wrote anything.
type staffStore struct {
	staff       map[string]*models.Staff
	lookupErr   error
	createCalls int
}

func emptyStore() *staffStore {
	return &staffStore{staff: map[string]*models.Staff{}}
}

func (s *staffStore) GetStaffByUsername(username string) (*models.Staff, error) {
	if s.lookupErr != nil {
		return nil, s.lookupErr
	}
	found, ok := s.staff[username]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return found, nil
}

func (s *staffStore) CreateStaff(staff *models.Staff) error {
	s.createCalls++
	staff.ID = uint(len(s.staff) + 1)
	s.staff[staff.Username] = staff
	return nil
}

func newStaffRouter(store *staffStore) (*gin.Engine, *utils.JwtUtil) {
	gin.SetMode(gin.TestMode)
	jwtUtil := &utils.JwtUtil{SecretKey: []byte("test-secret"), Expiration: time.Hour}
	handler := &handlers.StaffHandler{Service: &services.StaffService{Repo: store, JWT: jwtUtil}}
	router := gin.New()
	router.POST("/staff/create", handler.CreateStaff)
	router.POST("/staff/login", handler.LoginStaff)
	return router, jwtUtil
}

func post(t *testing.T, router *gin.Engine, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

// ---------- POST /staff/create ----------

func TestCreateThenLoginIssuesAUsableToken(t *testing.T) {
	router, jwtUtil := newStaffRouter(emptyStore())
	credentials := `{"username":"nurse01","password":"secret123","hospital":"A"}`

	created := post(t, router, "/staff/create", credentials)
	if created.Code != http.StatusCreated {
		t.Fatalf("create: got %d %s, want 201", created.Code, created.Body)
	}

	loggedIn := post(t, router, "/staff/login", credentials)
	if loggedIn.Code != http.StatusOK {
		t.Fatalf("login: got %d %s, want 200", loggedIn.Code, loggedIn.Body)
	}

	var body struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(loggedIn.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	claims, err := jwtUtil.ParseToken(body.Token)
	if err != nil {
		t.Fatalf("the issued token does not parse: %v", err)
	}
	if claims.Hospital != "A" {
		t.Errorf("got hospital %q, want \"A\"", claims.Hospital)
	}
}

func TestCreateRejectsTheSameUsernameTwice(t *testing.T) {
	router, _ := newStaffRouter(emptyStore())
	credentials := `{"username":"nurse01","password":"secret123","hospital":"A"}`
	post(t, router, "/staff/create", credentials)

	again := post(t, router, "/staff/create", credentials)
	if again.Code != http.StatusConflict {
		t.Errorf("got %d, want 409", again.Code)
	}
}

// A username already in the table must be rejected without a second insert.
func TestCreateStaffReturnsConflictForExistingUsername(t *testing.T) {
	store := emptyStore()
	store.staff["alice"] = &models.Staff{Username: "alice"}
	router, _ := newStaffRouter(store)

	response := post(t, router, "/staff/create", `{"username":"alice","password":"password","hospital":"Hospital A"}`)
	if response.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", response.Code, response.Body.String())
	}
	if store.createCalls != 0 {
		t.Error("an existing username must not reach the insert")
	}
}

// The binding tags are the only thing rejecting weak or malformed input, so they
// are worth pinning down.
func TestCreateValidatesTheRequestBody(t *testing.T) {
	for _, tt := range []struct {
		name string
		body string
	}{
		{"password below the minimum", `{"username":"nurse01","password":"short","hospital":"A"}`},
		{"username is not alphanumeric", `{"username":"nurse 01!","password":"secret123","hospital":"A"}`},
		{"hospital missing", `{"username":"nurse01","password":"secret123"}`},
		{"not json at all", `nonsense`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			router, _ := newStaffRouter(emptyStore())
			response := post(t, router, "/staff/create", tt.body)
			if response.Code != http.StatusBadRequest {
				t.Errorf("got %d, want 400", response.Code)
			}
			if strings.Contains(response.Body.String(), "createStaffRequest") {
				t.Errorf("the internal struct name leaked: %s", response.Body)
			}
		})
	}
}

// ---------- POST /staff/login ----------

func TestLoginRejectsAnUnknownUser(t *testing.T) {
	router, _ := newStaffRouter(emptyStore())

	response := post(t, router, "/staff/login", `{"username":"ghost","password":"secret123","hospital":"A"}`)
	if response.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", response.Code)
	}
}

func TestLoginStaffMapsErrorsToHTTPStatus(t *testing.T) {
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
			store := emptyStore()
			store.lookupErr = tc.lookupErr
			router, _ := newStaffRouter(store)

			response := post(t, router, "/staff/login", `{"username":"alice","password":"password","hospital":"Hospital A"}`)
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
			if store.createCalls != 0 {
				t.Error("login must not write to the staff table")
			}
		})
	}
}
