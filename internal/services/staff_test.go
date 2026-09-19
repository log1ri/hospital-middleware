package services

import (
	"errors"
	"testing"
	"time"

	"hospital-middleware/internal/models"
	"hospital-middleware/internal/utils"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// fakeStaffStore covers both halves of StaffService. A lookup returns, in order
// of precedence: an injected error, an explicit staff record, a blank record
// when found is set, or "not found". Writes are recorded rather than performed,
// so a test can assert that a code path never inserted anything.
type fakeStaffStore struct {
	staff       *models.Staff
	found       bool
	lookupErr   error
	createErr   error
	lookupCalls int
	created     *models.Staff
}

func (f *fakeStaffStore) GetStaffByUsername(string) (*models.Staff, error) {
	f.lookupCalls++
	if f.lookupErr != nil {
		return nil, f.lookupErr
	}
	if f.staff != nil {
		return f.staff, nil
	}
	if f.found {
		return &models.Staff{}, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeStaffStore) CreateStaff(staff *models.Staff) error {
	f.created = staff
	return f.createErr
}

// newLoginService builds a service whose stored hash comes from the real hashing
// code, so login is exercised end to end rather than against a stub.
func newLoginService(t *testing.T, password, hospital string) (*StaffService, *fakeStaffStore) {
	t.Helper()
	hash, err := utils.HashPassword(password)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	store := &fakeStaffStore{staff: &models.Staff{ID: 7, Username: "nurse", PasswordHash: hash, Hospital: hospital}}
	return &StaffService{
		Repo: store,
		JWT:  &utils.JwtUtil{SecretKey: []byte("test-secret"), Expiration: time.Hour},
	}, store
}

// ---------- CreateStaff ----------

func TestCreateStaffRejectsExistingUsernameBeforeInsert(t *testing.T) {
	store := &fakeStaffStore{found: true}
	service := StaffService{Repo: store}

	err := service.CreateStaff("alice", "password", "Hospital A")
	if !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("expected ErrUsernameTaken, got %v", err)
	}
	if store.lookupCalls != 1 || store.created != nil {
		t.Fatal("existing username should stop before inserting")
	}
}

func TestCreateStaffStopsOnLookupError(t *testing.T) {
	lookupErr := errors.New("database unavailable")
	store := &fakeStaffStore{lookupErr: lookupErr}
	service := StaffService{Repo: store}

	if err := service.CreateStaff("alice", "password", "Hospital A"); !errors.Is(err, lookupErr) {
		t.Fatalf("expected lookup error, got %v", err)
	}
	if store.created != nil {
		t.Fatal("staff was inserted after lookup failed")
	}
}

func TestCreateStaffMapsConcurrentDuplicate(t *testing.T) {
	store := &fakeStaffStore{createErr: gorm.ErrDuplicatedKey}
	service := StaffService{Repo: store}

	if err := service.CreateStaff("alice", "password", "Hospital A"); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("expected ErrUsernameTaken, got %v", err)
	}
	if store.lookupCalls != 1 || store.created == nil {
		t.Fatal("expected lookup followed by attempted insert")
	}
}

func TestCreateStaffPropagatesWriteError(t *testing.T) {
	writeErr := errors.New("database unavailable")
	store := &fakeStaffStore{createErr: writeErr}
	service := StaffService{Repo: store}

	if err := service.CreateStaff("alice", "password", "Hospital A"); !errors.Is(err, writeErr) {
		t.Fatalf("expected write error, got %v", err)
	}
}

func TestCreateStaffStoresHashedPassword(t *testing.T) {
	store := &fakeStaffStore{}
	service := StaffService{Repo: store}

	if err := service.CreateStaff("alice", "password", "Hospital A"); err != nil {
		t.Fatal(err)
	}
	if store.created == nil || store.created.Username != "alice" || store.created.Hospital != "Hospital A" {
		t.Fatalf("unexpected staff record: %+v", store.created)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(store.created.PasswordHash), []byte("password")); err != nil {
		t.Fatalf("password was not hashed correctly: %v", err)
	}
	if store.lookupCalls != 1 {
		t.Fatal("CreateStaff should check the username before inserting")
	}
}

// ---------- LoginStaff ----------

func TestLoginReturnsATokenCarryingTheStaffIdentity(t *testing.T) {
	service, store := newLoginService(t, "secret123", "A")

	token, err := service.LoginStaff("nurse", "secret123", "A")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if store.created != nil {
		t.Error("login must not write to the staff table")
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
	service, store := newLoginService(t, "secret123", "A")

	if _, err := service.LoginStaff("nurse", "wrong-password", "A"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("got %v, want ErrInvalidCredentials", err)
	}
	if store.created != nil {
		t.Error("a failed login must not write to the staff table")
	}
}

// Staff must not be able to sign in against a hospital they do not belong to,
// even with the right password.
func TestLoginRejectsAnotherHospital(t *testing.T) {
	service, _ := newLoginService(t, "secret123", "A")

	if _, err := service.LoginStaff("nurse", "secret123", "B"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestLoginTreatsAnUnknownUserAsBadCredentials(t *testing.T) {
	service := &StaffService{
		Repo: &fakeStaffStore{lookupErr: gorm.ErrRecordNotFound},
		JWT:  &utils.JwtUtil{SecretKey: []byte("test-secret"), Expiration: time.Hour},
	}

	if _, err := service.LoginStaff("ghost", "secret123", "A"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestLoginPropagatesLookupFailures(t *testing.T) {
	failure := errors.New("database is down")
	service := &StaffService{
		Repo: &fakeStaffStore{lookupErr: failure},
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
