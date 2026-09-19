package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"hospital-middleware/internal/models"
)

func Open(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
}

func Init(dsn string) (*gorm.DB, error) {
	database, err := Open(dsn)
	if err != nil {
		return nil, err
	}

	if err := database.AutoMigrate(
		&models.Staff{},
		&models.Patient{},
	); err != nil {
		return nil, err
	}

	return database, nil
}
