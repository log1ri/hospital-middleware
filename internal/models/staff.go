package models

// Staff represents a staff member in the system.
type Staff struct {
	ID           uint   `gorm:"primaryKey"`
	Username     string `gorm:"uniqueIndex;not null"`
	PasswordHash string
	Hospital     string
}
