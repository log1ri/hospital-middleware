package config

import (
	"testing"
	"time"
)

func TestLoadJWTExpirationInSeconds(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("JWT_EXPIRATION", "7200")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.JWTExpiration != 2*time.Hour {
		t.Fatalf("expected 2 hours, got %s", cfg.JWTExpiration)
	}
}

func TestLoadRejectsInvalidJWTExpiration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("JWT_SECRET", "test-secret")

	for _, value := range []string{"", "0", "-1", "2h", "9223372037"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("JWT_EXPIRATION", value)
			if _, err := Load(); err == nil {
				t.Fatalf("expected error for JWT_EXPIRATION=%q", value)
			}
		})
	}
}

func TestLoadHospitalABaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("JWT_EXPIRATION", "7200")
	t.Setenv("HOSPITAL_A_BASE_URL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HospitalABaseURL != defaultHospitalABaseURL {
		t.Fatalf("expected default Hospital A URL, got %q", cfg.HospitalABaseURL)
	}

	t.Setenv("HOSPITAL_A_BASE_URL", "http://localhost:8081")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HospitalABaseURL != "http://localhost:8081" {
		t.Fatalf("expected configured Hospital A URL, got %q", cfg.HospitalABaseURL)
	}
}
