package models_test

import (
	"database/sql/driver"
	"encoding/json"
	"testing"
	"time"

	"hospital-middleware/internal/models"
)

func date(t *testing.T, value string) *models.Date {
	t.Helper()
	parsed, err := time.Parse(models.DateLayout, value)
	if err != nil {
		t.Fatalf("parse %q: %v", value, err)
	}
	result := models.Date(parsed)
	return &result
}

func TestPatientMarshalsDateWithoutTimestamp(t *testing.T) {
	body, err := json.Marshal(models.Patient{HN: "A0001", DateOfBirth: date(t, "1990-01-05")})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got, want := string(body), `{"hn":"A0001","date_of_birth":"1990-01-05"}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestPatientOmitsMissingDate(t *testing.T) {
	body, err := json.Marshal(models.Patient{HN: "A0001"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got, want := string(body), `{"hn":"A0001"}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// The driver must receive a plain string: sending a time.Time would let Postgres
// convert a timestamp into a date and shift the day in non-UTC timezones.
func TestValueIsAPlainDateString(t *testing.T) {
	value, err := date(t, "1990-01-05").Value()
	if err != nil {
		t.Fatalf("value: %v", err)
	}
	if value != driver.Value("1990-01-05") {
		t.Errorf("got %#v, want \"1990-01-05\"", value)
	}
}

func TestNilDateIsStoredAsNull(t *testing.T) {
	value, err := driver.DefaultParameterConverter.ConvertValue((*models.Date)(nil))
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if value != nil {
		t.Errorf("got %#v, want nil", value)
	}
}

func TestScanReadsPostgresDate(t *testing.T) {
	var scanned models.Date
	if err := scanned.Scan(time.Date(1990, time.January, 5, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if scanned.String() != "1990-01-05" {
		t.Errorf("got %s, want 1990-01-05", scanned.String())
	}
}

func TestScanAcceptsANullColumn(t *testing.T) {
	var scanned models.Date
	if err := scanned.Scan(nil); err != nil {
		t.Errorf("a NULL date was rejected: %v", err)
	}
}

func TestScanRejectsAnUnexpectedColumnType(t *testing.T) {
	var scanned models.Date
	if err := scanned.Scan(12345); err == nil {
		t.Error("an integer was scanned into a Date without complaint")
	}
}
