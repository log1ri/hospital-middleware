package repositories

import (
	"context"

	"hospital-middleware/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// searchLimit caps how many patients one search can return.
const searchLimit = 100

type PatientRepository struct {
	DB *gorm.DB
}

// Search searches for patients in the database based on the provided filters.
func (r *PatientRepository) Search(ctx context.Context, hospital string, filters models.PatientFilters) ([]models.Patient, error) {
	// Start building the query with the hospital filter
	query := r.DB.WithContext(ctx).Where("hospital = ?", hospital)

	if filters.NationalID != "" {
		query = query.Where("national_id = ?", filters.NationalID)
	}
	if filters.PassportID != "" {
		query = query.Where("passport_id = ?", filters.PassportID)
	}
	if filters.FirstName != "" {
		query = query.Where("(first_name_th = ? OR first_name_en = ?)", filters.FirstName, filters.FirstName)
	}
	if filters.MiddleName != "" {
		query = query.Where("(middle_name_th = ? OR middle_name_en = ?)", filters.MiddleName, filters.MiddleName)
	}
	if filters.LastName != "" {
		query = query.Where("(last_name_th = ? OR last_name_en = ?)", filters.LastName, filters.LastName)
	}
	if !filters.DateOfBirth.IsZero() {
		query = query.Where("date_of_birth = ?", filters.DateOfBirth.Format(models.DateLayout))
	}
	if filters.PhoneNumber != "" {
		query = query.Where("phone_number = ?", filters.PhoneNumber)
	}
	if filters.Email != "" {
		query = query.Where("email = ?", filters.Email)
	}

	// Execute the query and return the results. An unfiltered search means "every
	// patient in this hospital", so the result is capped rather than unbounded.
	var patients []models.Patient
	err := query.Order("id").Limit(searchLimit).Find(&patients).Error
	return patients, err
}

// Upsert upserts a patient into the database, updating if it already exists.
func (r *PatientRepository) Upsert(ctx context.Context, patient *models.Patient) error {
	// Use GORM's Upsert feature to insert or update the patient record based on the unique constraint of hospital and HN
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "hospital"}, {Name: "hn"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"national_id", "passport_id",
			"first_name_th", "middle_name_th", "last_name_th",
			"first_name_en", "middle_name_en", "last_name_en",
			"date_of_birth", "phone_number", "email", "gender",
		}),
	}).Create(patient).Error
}
