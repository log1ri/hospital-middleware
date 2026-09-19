package repositories

import (
	"hospital-middleware/internal/models"

	"gorm.io/gorm"
)

type StaffRepository struct {
	DB *gorm.DB
}

func (r *StaffRepository) CreateStaff(staff *models.Staff) error {
	// Create a new staff member in the database
	return r.DB.Create(staff).Error
}

func (r *StaffRepository) GetStaffByUsername(username string) (*models.Staff, error) {
	// Fetch the staff member by username
	var staff models.Staff
	result := r.DB.Where("username = ?", username).Limit(1).Find(&staff)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &staff, nil
}
