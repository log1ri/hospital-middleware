package services

import (
	"testing"
	"time"

	"hospital-middleware/internal/models"
)

// matchesPatient is what decides whether a record fetched from the HIS survives
// the caller's filters. Hospital A answers ID lookups only, so every other
// filter in the assignment — name, date of birth, phone, email — is enforced
// here and nowhere else.
func TestMatchesPatient(t *testing.T) {
	nationalID := "1234567890123"
	passportID := "P1234567"
	dob := models.Date(time.Date(1990, 1, 5, 0, 0, 0, 0, time.UTC))

	patient := models.Patient{
		HN:           "A0001",
		NationalID:   &nationalID,
		PassportID:   &passportID,
		FirstNameTH:  "เดโม",
		MiddleNameTH: "กลาง",
		LastNameTH:   "ผู้ป่วย",
		FirstNameEN:  "Demo",
		MiddleNameEN: "Middle",
		LastNameEN:   "Patient",
		DateOfBirth:  &dob,
		PhoneNumber:  "0812345678",
		Email:        "demo@example.com",
	}

	for _, tt := range []struct {
		name    string
		filters models.PatientFilters
		want    bool
	}{
		{"no filters matches anyone", models.PatientFilters{}, true},

		// A filter has to match either language, which is the whole point of
		// comparing against both name columns.
		{"Thai first name", models.PatientFilters{FirstName: "เดโม"}, true},
		{"English first name", models.PatientFilters{FirstName: "Demo"}, true},
		{"Thai middle name", models.PatientFilters{MiddleName: "กลาง"}, true},
		{"English middle name", models.PatientFilters{MiddleName: "Middle"}, true},
		{"Thai last name", models.PatientFilters{LastName: "ผู้ป่วย"}, true},
		{"English last name", models.PatientFilters{LastName: "Patient"}, true},

		{"wrong first name is excluded", models.PatientFilters{FirstName: "Somchai"}, false},
		{"wrong last name is excluded", models.PatientFilters{LastName: "Jaidee"}, false},
		{"a name is matched whole, not by prefix", models.PatientFilters{FirstName: "Dem"}, false},
		{"matching is case sensitive", models.PatientFilters{FirstName: "demo"}, false},

		{"national id", models.PatientFilters{NationalID: nationalID}, true},
		{"wrong national id is excluded", models.PatientFilters{NationalID: "9999999999999"}, false},
		{"passport id", models.PatientFilters{PassportID: passportID}, true},
		{"wrong passport id is excluded", models.PatientFilters{PassportID: "X7654321"}, false},

		{"phone number", models.PatientFilters{PhoneNumber: "0812345678"}, true},
		{"wrong phone number is excluded", models.PatientFilters{PhoneNumber: "0800000000"}, false},
		{"email", models.PatientFilters{Email: "demo@example.com"}, true},
		{"wrong email is excluded", models.PatientFilters{Email: "other@example.com"}, false},

		{"date of birth", models.PatientFilters{DateOfBirth: time.Date(1990, 1, 5, 0, 0, 0, 0, time.UTC)}, true},
		{"wrong date of birth is excluded", models.PatientFilters{DateOfBirth: time.Date(1991, 1, 5, 0, 0, 0, 0, time.UTC)}, false},

		// Filters are combined with AND: every one of them has to hold.
		{"all filters agree", models.PatientFilters{FirstName: "Demo", LastName: "ผู้ป่วย", Email: "demo@example.com"}, true},
		{"one disagreeing filter rejects the patient", models.PatientFilters{FirstName: "Demo", LastName: "Jaidee"}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesPatient(patient, tt.filters); got != tt.want {
				t.Errorf("matchesPatient() = %v, want %v", got, tt.want)
			}
		})
	}
}

// A patient carrying only one identifier must not match a filter on the other,
// which is the case a nil pointer could silently get wrong.
func TestMatchesPatientWithMissingIdentifiers(t *testing.T) {
	passportOnly := "GB9876543"
	patient := models.Patient{HN: "A0006", PassportID: &passportOnly, FirstNameEN: "John"}

	if matchesPatient(patient, models.PatientFilters{NationalID: "1234567890123"}) {
		t.Error("a patient with no national ID must not match a national ID filter")
	}
	if !matchesPatient(patient, models.PatientFilters{PassportID: passportOnly}) {
		t.Error("the passport the patient does carry should still match")
	}
	if matchesPatient(patient, models.PatientFilters{DateOfBirth: time.Date(1969, 6, 15, 0, 0, 0, 0, time.UTC)}) {
		t.Error("a patient with no date of birth must not match a date filter")
	}
}
