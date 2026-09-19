package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"hospital-middleware/internal/utils"
)

func TestAuthenticatePassesStaffIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtUtil := &utils.JwtUtil{SecretKey: []byte("test-secret"), Expiration: 2 * time.Hour}
	token, err := jwtUtil.GenerateToken(42, "Hospital A")
	if err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.GET("/protected", Authenticate(jwtUtil), func(c *gin.Context) {
		staffID, ok := c.Get(StaffIDKey)
		if !ok || staffID != uint(42) {
			t.Errorf("unexpected staff ID: %v", staffID)
		}
		hospital, ok := c.Get(HospitalKey)
		if !ok || hospital != "Hospital A" {
			t.Errorf("unexpected hospital: %v", hospital)
		}
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", response.Code, response.Body.String())
	}
}

func TestAuthenticateRejectsInvalidTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtUtil := &utils.JwtUtil{SecretKey: []byte("test-secret"), Expiration: 2 * time.Hour}
	otherUtil := &utils.JwtUtil{SecretKey: []byte("other-secret"), Expiration: 2 * time.Hour}
	foreignToken, err := otherUtil.GenerateToken(42, "Hospital A")
	if err != nil {
		t.Fatal(err)
	}
	expiredToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, utils.StaffClaims{
		StaffID:  42,
		Hospital: "Hospital A",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
	}).SignedString(jwtUtil.SecretKey)
	if err != nil {
		t.Fatal(err)
	}
	missingHospitalToken, err := jwtUtil.GenerateToken(42, "")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name   string
		header string
	}{
		{"missing header", ""},
		{"wrong scheme", "Basic abc"},
		{"malformed header", "Bearer"},
		{"bad signature", "Bearer " + foreignToken},
		{"expired token", "Bearer " + expiredToken},
		{"missing hospital", "Bearer " + missingHospitalToken},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			router := gin.New()
			router.GET("/protected", Authenticate(jwtUtil), func(c *gin.Context) {
				called = true
				c.Status(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set("Authorization", tc.header)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized || called {
				t.Fatalf("expected blocked request with 401, got %d, handler called: %t", response.Code, called)
			}
		})
	}
}
