package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func devEnv() map[string]string {
	return map[string]string{
		"APP_ENV":      "development",
		"BASE_URL":     "http://localhost:8080",
		"DATABASE_URL": "postgres:///webapp_dev",
		"AUTH_MODE":    "dev",
	}
}

func TestLoadDevelopmentDefaults(t *testing.T) {
	c, err := Load(env(devEnv()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Addr != "127.0.0.1:8080" {
		t.Errorf("Addr = %q, want default", c.Addr)
	}
	if c.SessionIdle != 30*time.Minute || c.SessionAbsolute != 8*time.Hour {
		t.Errorf("session defaults = %v / %v", c.SessionIdle, c.SessionAbsolute)
	}
}

// Development sign-in needs no authentication, so it is refused in every
// environment except development (security review 2026-11-05, finding 2).
func TestLoadRejectsDevAuthOutsideDevelopment(t *testing.T) {
	for _, appEnv := range []string{"acceptance", "production"} {
		t.Run(appEnv, func(t *testing.T) {
			m := devEnv()
			m["APP_ENV"] = appEnv
			m["BASE_URL"] = "https://app.example.test"
			_, err := Load(env(m))
			if err == nil || !strings.Contains(err.Error(), "AUTH_MODE=dev is permitted only when APP_ENV=development") {
				t.Fatalf("want dev-auth rejection, got %v", err)
			}
		})
	}
	if _, err := Load(env(devEnv())); err != nil {
		t.Fatalf("development: unexpected error %v", err)
	}
}

func TestLoadRequiresTenantGUIDForEntra(t *testing.T) {
	m := devEnv()
	m["AUTH_MODE"] = "entra"
	m["ENTRA_CLIENT_ID"] = "c"
	m["SECRET_ENTRA_CLIENT_SECRET"] = "dev-only"
	m["ENTRA_TENANT_ID"] = "contoso.example"
	if _, err := Load(env(m)); err == nil || !strings.Contains(err.Error(), "ENTRA_TENANT_ID must be the tenant's GUID") {
		t.Fatalf("domain-name tenant: want rejection, got %v", err)
	}
	m["ENTRA_TENANT_ID"] = "00000000-0000-4000-8000-000000000001"
	if _, err := Load(env(m)); err != nil {
		t.Fatalf("GUID tenant: unexpected error %v", err)
	}
}

func TestLoadRequiresHTTPSOutsideDevelopment(t *testing.T) {
	m := devEnv()
	m["APP_ENV"] = "acceptance"
	_, err := Load(env(m))
	if err == nil || !strings.Contains(err.Error(), "https") {
		t.Fatalf("want https requirement, got %v", err)
	}
}

func TestLoadRejectsDatabaseRoleInProduction(t *testing.T) {
	m := devEnv()
	m["APP_ENV"] = "production"
	m["BASE_URL"] = "https://app.example.test"
	m["AUTH_MODE"] = "entra"
	m["ENTRA_TENANT_ID"], m["ENTRA_CLIENT_ID"] = "t", "c"
	m["DATABASE_ROLE"] = "webapp_app"
	_, err := Load(env(m))
	if err == nil || !strings.Contains(err.Error(), "DATABASE_ROLE") {
		t.Fatalf("want DATABASE_ROLE rejection, got %v", err)
	}
}

func TestLoadReportsAllProblems(t *testing.T) {
	_, err := Load(env(map[string]string{}))
	if err == nil {
		t.Fatal("want error")
	}
	for _, want := range []string{"APP_ENV", "DATABASE_URL", "BASE_URL", "AUTH_MODE"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %s: %v", want, err)
		}
	}
}

func TestReadSecretFromCredentialsDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "entra-client-secret"), []byte("s3cret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ReadSecret("entra-client-secret", Production, env(map[string]string{"CREDENTIALS_DIRECTORY": dir}))
	if err != nil || got != "s3cret" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestReadSecretEnvironmentFallbackNotInProduction(t *testing.T) {
	m := map[string]string{"SECRET_ENTRA_CLIENT_SECRET": "dev-only"}
	if got, err := ReadSecret("entra-client-secret", Development, env(m)); err != nil || got != "dev-only" {
		t.Fatalf("development: got %q, %v", got, err)
	}
	if _, err := ReadSecret("entra-client-secret", Production, env(m)); err == nil {
		t.Fatal("production must not read secrets from environment variables")
	}
}
