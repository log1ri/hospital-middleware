package services

import (
	"errors"
	"testing"
	"time"

	"hospital-middleware/internal/models"
	"hospital-middleware/internal/utils"

	"gorm.io/gorm"
)

// loginStaffStore returns a staff member whose hash was produced by the real
// hashing code, so login is exercised end to end rather than against a stub.
type loginStaffStore struct {
	staff *models.Staff
	err   error
}

func (s loginStaffStore) GetStaffByUsername(string) (*models.Staff, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.staff, nil
}

func (loginStaffStore) CreateStaff(*models.Staff) error {
	panic("CreateStaff must not be called during login")
}

func newLoginService(t *testing.T, password, hospital string) *StaffService {
	t.Helper()
	hash, err := utils.HashPassword(password)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	return &StaffService{
		Repo: loginStaffStore{staff: &models.Staff{ID: 7, Username: "nurse", PasswordHash: hash, Hospital: hospital}},
		JWT:  &utils.JwtUtil{SecretKey: []byte("test-secret"), Expiration: time.Hour},
	}
}

func TestLoginReturnsATokenCarryingTheStaffIdentity(t *testing.T) {
	service := newLoginService(t, "secret123", "A")

	token, err := service.LoginStaff("nurse", "secret123", "A")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	claims, err := service.JWT.ParseToken(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if claims.StaffID != 7 || claims.Hospital != "A" {
		t.Errorf("got staff %d at %q, want 7 at \"A\"", claims.StaffID, claims.Hospital)
	}
}

func TestLoginRejectsTheWrongPassword(t *testing.T) {
	service := newLoginService(t, "secret123", "A")

	if _, err := service.LoginStaff("nurse", "wrong-password", "A"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("got %v, want ErrInvalidCredentials", err)
	}
}

// Staff must not be able to sign in against a hospital they do not belong to,
// even with the right password.
func TestLoginRejectsAnotherHospital(t *testing.T) {
	service := newLoginService(t, "secret123", "A")

	if _, err := service.LoginStaff("nurse", "secret123", "B"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestLoginTreatsAnUnknownUserAsBadCredentials(t *testing.T) {
	service := &StaffService{
		Repo: loginStaffStore{err: gorm.ErrRecordNotFound},
		JWT:  &utils.JwtUtil{SecretKey: []byte("test-secret"), Expiration: time.Hour},
	}

	if _, err := service.LoginStaff("ghost", "secret123", "A"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestLoginPropagatesLookupFailures(t *testing.T) {
	failure := errors.New("database is down")
	service := &StaffService{
		Repo: loginStaffStore{err: failure},
		JWT:  &utils.JwtUtil{SecretKey: []byte("test-secret"), Expiration: time.Hour},
	}

	_, err := service.LoginStaff("nurse", "secret123", "A")
	if !errors.Is(err, failure) {
		t.Errorf("got %v, want the underlying failure", err)
	}
	if errors.Is(err, ErrInvalidCredentials) {
		t.Error("a database outage was reported as bad credentials")
	}
}
