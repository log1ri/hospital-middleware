package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"hospital-middleware/internal/clients"
	"hospital-middleware/internal/models"
)

var ErrPatientNotFound = clients.ErrPatientNotFound
var ErrUnsupportedHospital = errors.New("unsupported hospital")
var ErrPatientStore = errors.New("patient storage failed")
var ErrInvalidPatientResponse = errors.New("invalid Hospital A patient response")

type PatientSearcher interface {
	Search(ctx context.Context, id string) (json.RawMessage, error)
}

type PatientStore interface {
	Search(ctx context.Context, hospital string, filters models.PatientFilters) ([]models.Patient, error)
	Upsert(ctx context.Context, patient *models.Patient) error
}

type PatientService struct {
	Repo      PatientStore
	HospitalA PatientSearcher
}

func (s *PatientService) Search(ctx context.Context, hospital string, filters models.PatientFilters) ([]models.Patient, error) {
	// Currently, we only support searching patients from Hospital A.
	if hospital != "A" {
		return nil, ErrUnsupportedHospital
	}

	//=======First, check if the patient exists in the local database cache.=======
	cached, err := s.Repo.Search(ctx, hospital, filters)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrPatientStore, err)
	}
	if len(cached) > 0 {
		return cached, nil
	}

	//=======If not found in the cache, query Hospital A's HIS API.=======
	id := filters.NationalID
	if id == "" {
		id = filters.PassportID
	}
	if id == "" {
		// Hospital A can only be searched by ID, so name, date, phone and email
		// searches are answered from our own data alone. No match is an empty
		// list, not an error.
		return cached, nil
	}

	//Query Hospital A's HIS API for the patient.
	payload, err := s.HospitalA.Search(ctx, id)
	if err != nil {
		return nil, err
	}

	// Unmarshal the response into a temporary struct to validate and extract the patient data.
	var incoming struct {
		models.Patient
		PatientHN   string `json:"patient_hn"`
		DateOfBirth string `json:"date_of_birth"`
	}
	if err := json.Unmarshal(payload, &incoming); err != nil {
		return nil, ErrInvalidPatientResponse
	}

	// Validate the incoming patient data before it is trusted or stored.
	patient := incoming.Patient
	patient.HN = incoming.PatientHN
	patient.NationalID = omitBlank(patient.NationalID)
	patient.PassportID = omitBlank(patient.PassportID)
	if incoming.DateOfBirth != "" {
		date, err := time.Parse(models.DateLayout, incoming.DateOfBirth)
		if err != nil {
			return nil, ErrInvalidPatientResponse
		}
		birthDate := models.Date(date)
		patient.DateOfBirth = &birthDate
	}
	if patient.HN == "" {
		return nil, ErrInvalidPatientResponse
	}

	// Hospital A must answer with the patient we asked for. Storing somebody else
	// would hand staff another patient's record under the ID they searched for.
	returnedID := patient.NationalID
	if filters.NationalID == "" {
		returnedID = patient.PassportID
	}
	if !matchesOptionalID(id, returnedID) {
		return nil, ErrInvalidPatientResponse
	}

	// Store before filtering: the record is valid whether or not it happens to
	// match the other filters the staff typed.
	patient.Hospital = hospital
	if err := s.Repo.Upsert(ctx, &patient); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrPatientStore, err)
	}

	if !matchesPatient(patient, filters) {
		return nil, ErrPatientNotFound
	}
	return []models.Patient{patient}, nil
}

// omitBlank turns an empty or whitespace-only value into a NULL, so that "no
// passport" is stored as the absence of a value rather than as an empty string.
func omitBlank(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func matchesOne(expected string, values ...string) bool {
	if expected == "" {
		return true
	}
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func matchesOptionalID(expected string, actual *string) bool {
	return expected == "" || actual != nil && *actual == expected
}

func matchesDate(expected time.Time, actual *models.Date) bool {
	return expected.IsZero() || actual != nil && actual.String() == expected.Format(models.DateLayout)
}

func matchesPatient(patient models.Patient, f models.PatientFilters) bool {
	return matchesOptionalID(f.NationalID, patient.NationalID) &&
		matchesOptionalID(f.PassportID, patient.PassportID) &&
		matchesOne(f.FirstName, patient.FirstNameTH, patient.FirstNameEN) &&
		matchesOne(f.MiddleName, patient.MiddleNameTH, patient.MiddleNameEN) &&
		matchesOne(f.LastName, patient.LastNameTH, patient.LastNameEN) &&
		matchesDate(f.DateOfBirth, patient.DateOfBirth) &&
		matchesOne(f.PhoneNumber, patient.PhoneNumber) &&
		matchesOne(f.Email, patient.Email)
}
