package repositories_test

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"hospital-middleware/internal/db"
	"hospital-middleware/internal/models"
	"hospital-middleware/internal/repositories"
)

// newStaffRepo talks to the Postgres from docker-compose. Usernames are unique
// per run so parallel runs and leftovers never collide.
func newStaffRepo(t *testing.T) (*repositories.StaffRepository, string) {
	t.Helper()
	_ = godotenv.Load("../../.env")
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is not set; start docker compose to run database tests")
	}
	database, err := db.Init(dsn)
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}
	username := fmt.Sprintf("test%d", time.Now().UnixNano())
	t.Cleanup(func() {
		database.Exec("DELETE FROM staffs WHERE username LIKE 'test%'")
	})
	return &repositories.StaffRepository{DB: database}, username
}

func TestCreateStaffThenFindItBack(t *testing.T) {
	repo, username := newStaffRepo(t)
	staff := &models.Staff{Username: username, PasswordHash: "hashed", Hospital: "A"}

	if err := repo.CreateStaff(staff); err != nil {
		t.Fatalf("create: %v", err)
	}
	if staff.ID == 0 {
		t.Error("the generated ID was not written back to the struct")
	}

	found, err := repo.GetStaffByUsername(username)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if found.Username != username || found.Hospital != "A" || found.PasswordHash != "hashed" {
		t.Errorf("the row did not round trip: %+v", found)
	}
}

// The service relies on this exact error to tell "free username" from "database
// broke", so it is part of the contract rather than an implementation detail.
func TestGetStaffByUsernameReportsRecordNotFound(t *testing.T) {
	repo, username := newStaffRepo(t)

	_, err := repo.GetStaffByUsername(username + "-nobody")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("got %v, want gorm.ErrRecordNotFound", err)
	}
}

func TestCreateStaffRejectsADuplicateUsername(t *testing.T) {
	repo, username := newStaffRepo(t)
	if err := repo.CreateStaff(&models.Staff{Username: username, PasswordHash: "hashed", Hospital: "A"}); err != nil {
		t.Fatalf("create: %v", err)
	}

	err := repo.CreateStaff(&models.Staff{Username: username, PasswordHash: "other", Hospital: "B"})
	if err == nil {
		t.Fatal("the duplicate username was accepted")
	}
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Errorf("got %v, want gorm.ErrDuplicatedKey", err)
	}
}
