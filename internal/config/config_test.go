package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setValidEnv sets every required variable to something usable, so each test
// only has to vary the one it is actually about. Adding a new required variable
// means touching this helper rather than every test.
func setValidEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("JWT_EXPIRATION", "7200")
	t.Setenv("HOSPITAL_A_BASE_URL", "http://127.0.0.1:8081")
}

// godotenv abandons the whole file on the first malformed line, so a single
// dropped "=" would blank every value. That must fail loudly, not silently.
func TestLoadRejectsMalformedEnvFile(t *testing.T) {
	setValidEnv(t)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("GOOD=1\nBROKEN_NO_EQUALS\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	if _, err := Load(); err == nil {
		t.Fatal("expected an error for a malformed .env")
	}
}

// Containers get their values from the real environment and ship no .env, so a
// missing file has to stay fine.
func TestLoadWithoutEnvFile(t *testing.T) {
	setValidEnv(t)
	t.Chdir(t.TempDir())

	if _, err := Load(); err != nil {
		t.Fatalf("a missing .env must not be an error: %v", err)
	}
}

func TestLoadJWTExpirationInSeconds(t *testing.T) {
	setValidEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.JWTExpiration != 2*time.Hour {
		t.Fatalf("expected 2 hours, got %s", cfg.JWTExpiration)
	}
}

func TestLoadRejectsInvalidJWTExpiration(t *testing.T) {
	setValidEnv(t)

	for _, value := range []string{"", "0", "-1", "2h", "9223372037"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("JWT_EXPIRATION", value)
			if _, err := Load(); err == nil {
				t.Fatalf("expected error for JWT_EXPIRATION=%q", value)
			}
		})
	}
}

func TestLoadAppPort(t *testing.T) {
	setValidEnv(t)
	t.Setenv("APP_PORT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AppPort != defaultAppPort {
		t.Fatalf("expected default app port, got %q", cfg.AppPort)
	}

	t.Setenv("APP_PORT", "9090")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AppPort != "9090" {
		t.Fatalf("expected configured app port, got %q", cfg.AppPort)
	}

	for _, value := range []string{"0", "-1", "70000", "http"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("APP_PORT", value)
			if _, err := Load(); err == nil {
				t.Fatalf("expected error for APP_PORT=%q", value)
			}
		})
	}
}

func TestLoadAppEnv(t *testing.T) {
	setValidEnv(t)
	t.Setenv("APP_ENV", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AppEnv != EnvProduction {
		t.Fatalf("expected %s by default, got %q", EnvProduction, cfg.AppEnv)
	}

	t.Setenv("APP_ENV", EnvDevelopment)
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AppEnv != EnvDevelopment {
		t.Fatalf("expected %s, got %q", EnvDevelopment, cfg.AppEnv)
	}

	for _, value := range []string{"dev", "prod", "Production", "release"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("APP_ENV", value)
			if _, err := Load(); err == nil {
				t.Fatalf("expected error for APP_ENV=%q", value)
			}
		})
	}
}

// An unset Hospital A URL must fail rather than fall back to a default: a
// default would quietly send real requests to the real hospital.
func TestLoadRequiresHospitalABaseURL(t *testing.T) {
	setValidEnv(t)
	t.Setenv("HOSPITAL_A_BASE_URL", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected an error when HOSPITAL_A_BASE_URL is unset")
	}

	t.Setenv("HOSPITAL_A_BASE_URL", "http://localhost:8081")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HospitalABaseURL != "http://localhost:8081" {
		t.Fatalf("expected configured Hospital A URL, got %q", cfg.HospitalABaseURL)
	}
}
