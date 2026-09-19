package services

import (
	"errors"

	"hospital-middleware/internal/models"
	"hospital-middleware/internal/utils"

	"gorm.io/gorm"
)

var ErrUsernameTaken = errors.New("username already exists")
var ErrInvalidCredentials = errors.New("invalid credentials")

type StaffStore interface {
	GetStaffByUsername(username string) (*models.Staff, error)
	CreateStaff(staff *models.Staff) error
}

type StaffService struct {
	Repo StaffStore
	JWT  *utils.JwtUtil
}

func (s *StaffService) GetStaffByUsername(username string) (*models.Staff, error) {
	return s.Repo.GetStaffByUsername(username)
}

func (s *StaffService) CreateStaff(username, password, hospital string) error {
	// Check if the username already exists
	_, err := s.Repo.GetStaffByUsername(username)
	if err == nil {
		return ErrUsernameTaken
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// Hash the password before storing it
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	// Create a new staff member
	staff := &models.Staff{
		Username:     username,
		PasswordHash: hashedPassword,
		Hospital:     hospital,
	}

	// Insert the new staff member into the database
	err = s.Repo.CreateStaff(staff)
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrUsernameTaken
	}
	return err
}

func (s *StaffService) LoginStaff(username, password, hospital string) (string, error) {
	// Fetch the staff member by username
	staff, err := s.Repo.GetStaffByUsername(username)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", err
	}

	// Check if the provided password matches the stored password hash
	isValidPassword := utils.CheckPasswordHash(password, staff.PasswordHash)

	if staff.Hospital != hospital || !isValidPassword {
		return "", ErrInvalidCredentials
	}

	token, err := s.JWT.GenerateToken(staff.ID, staff.Hospital)
	if err != nil {
		return "", err
	}

	return token, nil

}
