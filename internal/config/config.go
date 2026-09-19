package config

import (
	"errors"
	"fmt"
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
	AppPort          string
	AppEnv           string
}

// The two environments the app knows about. Everything that should differ
// between a developer's machine and a deployment is derived from this, so no
// caller has to know about framework-specific switches like GIN_MODE.
const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
)

const defaultAppPort = "8080"

// Production by default: a missing APP_ENV must not be what turns on debug
// output in a deployment.
const defaultAppEnv = EnvProduction

func Load() (Config, error) {
	// A missing .env is normal: in a container the values come from the real
	// environment. A malformed one is not — godotenv abandons the whole file on
	// the first bad line, so one dropped "=" would silently blank every value.
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

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

	// Deliberately has no default. A default would mean that failing to set it —
	// or failing to load .env at all — silently points the middleware at the real
	// hospital instead of a local stand-in.
	hospitalA := os.Getenv("HOSPITAL_A_BASE_URL")
	if hospitalA == "" {
		return Config{}, errors.New("HOSPITAL_A_BASE_URL is required")
	}

	port, err := appPort()
	if err != nil {
		return Config{}, err
	}

	environment, err := appEnv()
	if err != nil {
		return Config{}, err
	}

	return Config{
		DatabaseURL:      dsn,
		JWTSecret:        secret,
		JWTExpiration:    time.Duration(expirationSeconds) * time.Second,
		HospitalABaseURL: hospitalA,
		AppPort:          port,
		AppEnv:           environment,
	}, nil
}

func appEnv() (string, error) {
	switch value := os.Getenv("APP_ENV"); value {
	case "":
		return defaultAppEnv, nil
	case EnvDevelopment, EnvProduction:
		return value, nil
	default:
		return "", errors.New("APP_ENV must be " + EnvDevelopment + " or " + EnvProduction)
	}
}

// appPort rejects a bad value up front, so a typo fails with a config error at
// startup rather than an opaque listen error further along.
func appPort() (string, error) {
	value := os.Getenv("APP_PORT")
	if value == "" {
		return defaultAppPort, nil
	}
	if port, err := strconv.Atoi(value); err != nil || port < 1 || port > 65535 {
		return "", errors.New("APP_PORT must be a port number between 1 and 65535")
	}
	return value, nil
}
