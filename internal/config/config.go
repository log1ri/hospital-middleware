package config

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL      string
	JWTSecret        string
	JWTExpiration    time.Duration
	HospitalABaseURL string
}

const defaultHospitalABaseURL = "https://hospital-a.api.co.th"

func Load() (Config, error) {
	_ = godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	expirationSeconds, err := strconv.ParseInt(os.Getenv("JWT_EXPIRATION"), 10, 64)
	if err != nil || expirationSeconds <= 0 || expirationSeconds > int64(^uint64(0)>>1)/int64(time.Second) {
		return Config{}, errors.New("JWT_EXPIRATION must be a positive number of seconds")
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}

	return Config{
		DatabaseURL:      dsn,
		JWTSecret:        secret,
		JWTExpiration:    time.Duration(expirationSeconds) * time.Second,
		HospitalABaseURL: hospitalABaseURL(),
	}, nil
}

func hospitalABaseURL() string {
	if value := os.Getenv("HOSPITAL_A_BASE_URL"); value != "" {
		return value
	}
	return defaultHospitalABaseURL
}
