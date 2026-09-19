package repositories_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"hospital-middleware/internal/db"
	"hospital-middleware/internal/models"
	"hospital-middleware/internal/repositories"
)

// newTestRepo talks to the Postgres from docker-compose. Every test gets its own
// hospital name, so rows never collide and cleanup is a single scoped delete.
func newTestRepo(t *testing.T) (*repositories.PatientRepository, string) {
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
	hospital := fmt.Sprintf("TEST-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		database.Exec("DELETE FROM patients WHERE hospital LIKE 'TEST-%'")
	})
	return &repositories.PatientRepository{DB: database}, hospital
}

func text(value string) *string { return &value }

func date(t *testing.T, value string) *models.Date {
	t.Helper()
	parsed, err := time.Parse(models.DateLayout, value)
	if err != nil {
		t.Fatalf("parse %q: %v", value, err)
	}
	result := models.Date(parsed)
	return &result
}

func seed(t *testing.T, repo *repositories.PatientRepository, patients ...models.Patient) {
	t.Helper()
	for i := range patients {
		if err := repo.Upsert(context.Background(), &patients[i]); err != nil {
			t.Fatalf("seed %s: %v", patients[i].HN, err)
		}
	}
}

func hns(patients []models.Patient) []string {
	result := []string{}
	for _, patient := range patients {
		result = append(result, patient.HN)
	}
	return result
}

func TestSearchFilters(t *testing.T) {
	repo, hospital := newTestRepo(t)
	somchai := models.Patient{
		Hospital: hospital, HN: "HN001",
		NationalID: text("1111111111111"), PassportID: text("P1111"),
		FirstNameTH: "สมชาย", LastNameTH: "ใจดี",
		FirstNameEN: "Somchai", LastNameEN: "Jaidee",
		DateOfBirth: date(t, "1990-01-05"),
		PhoneNumber: "0811111111", Email: "somchai@example.com", Gender: "M",
	}
	jane := models.Patient{
		Hospital: hospital, HN: "HN002",
		PassportID:  text("P2222"),
		FirstNameEN: "Jane", LastNameEN: "Doe",
		DateOfBirth: date(t, "1985-12-31"),
		PhoneNumber: "0822222222", Email: "jane@example.com", Gender: "F",
	}
	// Same national ID, different hospital: must never show up in the results.
	other := models.Patient{
		Hospital: hospital + "-OTHER", HN: "HN001",
		NationalID: text("1111111111111"), FirstNameEN: "Somchai",
	}
	seed(t, repo, somchai, jane, other)

	for _, tt := range []struct {
		name    string
		filters models.PatientFilters
		wantHNs []string
	}{
		{"no filters returns the hospital's patients", models.PatientFilters{}, []string{"HN001", "HN002"}},
		{"national id", models.PatientFilters{NationalID: "1111111111111"}, []string{"HN001"}},
		{"passport id", models.PatientFilters{PassportID: "P2222"}, []string{"HN002"}},
		{"thai first name", models.PatientFilters{FirstName: "สมชาย"}, []string{"HN001"}},
		{"english first name", models.PatientFilters{FirstName: "Somchai"}, []string{"HN001"}},
		{"english last name", models.PatientFilters{LastName: "Doe"}, []string{"HN002"}},
		{"phone number", models.PatientFilters{PhoneNumber: "0822222222"}, []string{"HN002"}},
		{"email", models.PatientFilters{Email: "somchai@example.com"}, []string{"HN001"}},
		{"date of birth", models.PatientFilters{DateOfBirth: parseDate(t, "1990-01-05")}, []string{"HN001"}},
		{"filters combine with AND", models.PatientFilters{FirstName: "Somchai", Email: "jane@example.com"}, []string{}},
		{"unknown value matches nothing", models.PatientFilters{Email: "nobody@example.com"}, []string{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			found, err := repo.Search(context.Background(), hospital, tt.filters)
			if err != nil {
				t.Fatalf("search: %v", err)
			}
			if got := fmt.Sprint(hns(found)); got != fmt.Sprint(tt.wantHNs) {
				t.Errorf("got %s, want %v", got, tt.wantHNs)
			}
		})
	}
}

func parseDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(models.DateLayout, value)
	if err != nil {
		t.Fatalf("parse %q: %v", value, err)
	}
	return parsed
}

// A date must survive the round trip unshifted, whatever timezone the server runs in.
func TestDateOfBirthRoundTrip(t *testing.T) {
	repo, hospital := newTestRepo(t)
	seed(t, repo, models.Patient{Hospital: hospital, HN: "HN001", DateOfBirth: date(t, "1990-01-05")})

	found, err := repo.Search(context.Background(), hospital, models.PatientFilters{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(found) != 1 || found[0].DateOfBirth == nil {
		t.Fatalf("expected one patient with a birth date, got %+v", found)
	}
	if got := found[0].DateOfBirth.String(); got != "1990-01-05" {
		t.Errorf("got %s, want 1990-01-05", got)
	}
}

func TestSearchReadsNullIdentifiersBack(t *testing.T) {
	repo, hospital := newTestRepo(t)
	seed(t, repo, models.Patient{Hospital: hospital, HN: "HN001", PassportID: text("P1")})

	found, err := repo.Search(context.Background(), hospital, models.PatientFilters{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("expected one patient, got %d", len(found))
	}
	if found[0].NationalID != nil {
		t.Errorf("expected a NULL national id, got %q", *found[0].NationalID)
	}
	if found[0].PassportID == nil || *found[0].PassportID != "P1" {
		t.Errorf("passport id did not round trip: %v", found[0].PassportID)
	}
}

func TestUpsertUpdatesInsteadOfDuplicating(t *testing.T) {
	repo, hospital := newTestRepo(t)
	seed(t, repo, models.Patient{
		Hospital: hospital, HN: "HN001",
		NationalID: text("1111111111111"), Email: "before@example.com",
	})

	updated := models.Patient{
		Hospital: hospital, HN: "HN001",
		NationalID: text("1111111111111"), Email: "after@example.com",
		PhoneNumber: "0899999999",
	}
	if err := repo.Upsert(context.Background(), &updated); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	found, err := repo.Search(context.Background(), hospital, models.PatientFilters{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("expected the row to be updated, got %d rows", len(found))
	}
	if found[0].Email != "after@example.com" || found[0].PhoneNumber != "0899999999" {
		t.Errorf("row was not updated: %+v", found[0])
	}
}

// Two patients without a national ID must coexist: NULL never collides in an index.
func TestNullIdentifiersDoNotCollide(t *testing.T) {
	repo, hospital := newTestRepo(t)
	seed(t, repo,
		models.Patient{Hospital: hospital, HN: "HN001"},
		models.Patient{Hospital: hospital, HN: "HN002"},
	)

	found, err := repo.Search(context.Background(), hospital, models.PatientFilters{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(found) != 2 {
		t.Fatalf("expected both patients, got %d", len(found))
	}
}
