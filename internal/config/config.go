// Package config loads and validates all configuration at startup.
// The application refuses to start with missing or invalid configuration
// (Web-App Development Stack v2.1, Section 11).
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Env is the operating environment (Web-App Development Stack v2.1, Section 16).
type Env string

const (
	Development Env = "development"
	Acceptance  Env = "acceptance"
	Production  Env = "production"
)

// Authentication modes.
const (
	AuthEntra = "entra" // Microsoft Entra ID through OpenID Connect
	AuthDev   = "dev"   // Development sign-in as a synthetic user; only when APP_ENV=development
)

// guidPattern matches a GUID such as an Entra ID tenant ID.
var guidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Config is the complete application configuration.
type Config struct {
	Env             Env
	Addr            string // listen address, e.g. 127.0.0.1:8080
	BaseURL         string // public URL, e.g. https://procurement.deltra.example
	DatabaseURL     string // libpq-style connection string or URL
	DatabaseRole    string // optional role to SET after connecting (development and tests)
	AuthMode        string
	Entra           EntraConfig
	SessionIdle     time.Duration
	SessionAbsolute time.Duration
	LogLevel        slog.Level
}

// EntraConfig holds Microsoft Entra ID application registration settings.
type EntraConfig struct {
	TenantID     string
	ClientID     string
	ClientSecret string // secret: read from a systemd credential, never from source control
	RedirectURL  string
}

// Load reads configuration using getenv (normally os.Getenv) and validates it.
func Load(getenv func(string) string) (Config, error) {
	var errs []error
	c := Config{
		Env:          Env(strings.ToLower(getenv("APP_ENV"))),
		Addr:         getenv("APP_ADDR"),
		BaseURL:      strings.TrimRight(getenv("BASE_URL"), "/"),
		DatabaseURL:  getenv("DATABASE_URL"),
		DatabaseRole: getenv("DATABASE_ROLE"),
		AuthMode:     strings.ToLower(getenv("AUTH_MODE")),
	}

	switch c.Env {
	case Development, Acceptance, Production:
	default:
		errs = append(errs, fmt.Errorf("APP_ENV must be development, acceptance, or production (got %q)", c.Env))
	}
	if c.Addr == "" {
		c.Addr = "127.0.0.1:8080"
	}
	if c.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if c.DatabaseRole != "" && c.Env == Production {
		errs = append(errs, errors.New("DATABASE_ROLE must not be set in production; connect as the application role"))
	}

	u, err := url.Parse(c.BaseURL)
	switch {
	case c.BaseURL == "":
		errs = append(errs, errors.New("BASE_URL is required"))
	case err != nil || u.Host == "":
		errs = append(errs, fmt.Errorf("BASE_URL is not a valid URL: %q", c.BaseURL))
	case c.Env != Development && u.Scheme != "https":
		errs = append(errs, errors.New("BASE_URL must use https outside development"))
	}

	switch c.AuthMode {
	case AuthEntra:
		c.Entra = EntraConfig{
			TenantID:    getenv("ENTRA_TENANT_ID"),
			ClientID:    getenv("ENTRA_CLIENT_ID"),
			RedirectURL: c.BaseURL + "/auth/callback",
		}
		if c.Entra.TenantID == "" || c.Entra.ClientID == "" {
			errs = append(errs, errors.New("ENTRA_TENANT_ID and ENTRA_CLIENT_ID are required when AUTH_MODE=entra"))
		} else if !guidPattern.MatchString(c.Entra.TenantID) {
			// Users are matched on the tid claim, which is always a GUID.
			errs = append(errs, errors.New("ENTRA_TENANT_ID must be the tenant's GUID, not a domain name"))
		}
		secret, err := ReadSecret("entra-client-secret", c.Env, getenv)
		if err != nil {
			errs = append(errs, err)
		}
		c.Entra.ClientSecret = secret
	case AuthDev:
		// Development sign-in needs no authentication, so it is refused in
		// acceptance as well as production.
		if c.Env != Development {
			errs = append(errs, fmt.Errorf("AUTH_MODE=dev is permitted only when APP_ENV=development (got %q)", c.Env))
		}
	default:
		errs = append(errs, fmt.Errorf("AUTH_MODE must be entra or dev (got %q)", c.AuthMode))
	}

	c.SessionIdle = durationOr(getenv("SESSION_IDLE"), 30*time.Minute, "SESSION_IDLE", &errs)
	c.SessionAbsolute = durationOr(getenv("SESSION_ABSOLUTE"), 8*time.Hour, "SESSION_ABSOLUTE", &errs)
	if c.SessionIdle > c.SessionAbsolute {
		errs = append(errs, errors.New("SESSION_IDLE must not exceed SESSION_ABSOLUTE"))
	}

	if lvl := getenv("LOG_LEVEL"); lvl != "" {
		if err := c.LogLevel.UnmarshalText([]byte(lvl)); err != nil {
			errs = append(errs, fmt.Errorf("LOG_LEVEL: %w", err))
		}
	}

	return c, errors.Join(errs...)
}

// ReadSecret returns the named secret. In every environment it is read from
// the systemd credentials directory when CREDENTIALS_DIRECTORY is set
// (Web-App Development Stack v2.1, Section 10.8). Outside production only,
// it may instead come from the environment variable SECRET_<NAME>
// (for example SECRET_ENTRA_CLIENT_SECRET), holding a development-only value.
func ReadSecret(name string, env Env, getenv func(string) string) (string, error) {
	if dir := getenv("CREDENTIALS_DIRECTORY"); dir != "" {
		b, err := readCredential(dir, filepath.Base(name))
		if err == nil {
			return strings.TrimSpace(string(b)), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("reading credential %s: %w", name, err)
		}
	}
	if env != Production {
		key := "SECRET_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
		if v := getenv(key); v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("secret %s not found in the systemd credentials directory", name)
}

// readCredential reads one file from the credentials directory through an
// os.Root, so the name cannot resolve outside that directory.
func readCredential(dir, name string) ([]byte, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	return root.ReadFile(name)
}

func durationOr(s string, def time.Duration, name string, errs *[]error) time.Duration {
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		*errs = append(*errs, fmt.Errorf("%s must be a positive duration such as 30m (got %q)", name, s))
		return def
	}
	return d
}
