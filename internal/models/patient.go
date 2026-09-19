package models

import "time"

type Patient struct {
	ID           uint    `gorm:"primaryKey" json:"-"`
	Hospital     string  `gorm:"not null;uniqueIndex:idx_patient_hospital_hn;index:idx_patient_hospital_national;index:idx_patient_hospital_passport" json:"-"`
	HN           string  `gorm:"not null;uniqueIndex:idx_patient_hospital_hn" json:"hn"`
	NationalID   *string `gorm:"index:idx_patient_hospital_national" json:"national_id,omitempty"`
	PassportID   *string `gorm:"index:idx_patient_hospital_passport" json:"passport_id,omitempty"`
	FirstNameTH  string  `json:"first_name_th,omitempty"`
	MiddleNameTH string  `json:"middle_name_th,omitempty"`
	LastNameTH   string  `json:"last_name_th,omitempty"`
	FirstNameEN  string  `json:"first_name_en,omitempty"`
	MiddleNameEN string  `json:"middle_name_en,omitempty"`
	LastNameEN   string  `json:"last_name_en,omitempty"`
	DateOfBirth  *Date   `gorm:"type:date" json:"date_of_birth,omitempty"`
	PhoneNumber  string  `json:"phone_number,omitempty"`
	Email        string  `json:"email,omitempty"`
	Gender       string  `json:"gender,omitempty"`
}

type PatientFilters struct {
	NationalID  string    `form:"national_id"`
	PassportID  string    `form:"passport_id"`
	FirstName   string    `form:"first_name"`
	MiddleName  string    `form:"middle_name"`
	LastName    string    `form:"last_name"`
	DateOfBirth time.Time `form:"date_of_birth" time_format:"2006-01-02"`
	PhoneNumber string    `form:"phone_number"`
	Email       string    `form:"email"`
}
