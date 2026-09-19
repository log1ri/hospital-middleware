package services

import (
	"errors"
	"testing"

	"hospital-middleware/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type fakeStaffStore struct {
	lookupCalls int
	found       bool
	lookupErr   error
	createErr   error
	created     *models.Staff
}

func (f *fakeStaffStore) GetStaffByUsername(string) (*models.Staff, error) {
	f.lookupCalls++
	if f.lookupErr != nil {
		return nil, f.lookupErr
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
