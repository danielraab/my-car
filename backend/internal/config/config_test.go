package config

import (
	"fmt"
	"log/slog"
	"net/netip"
	"reflect"
	"strings"
	"testing"
)

func setAllEnv(t *testing.T) {
	t.Helper()
	values := map[string]string{
		"PORT":               "8080",
		"DATABASE_URL":       "postgres://mycars:mycars@localhost:5432/mycars?sslmode=disable",
		"SMTP_FROM":          "admin@example.com",
		"SMTP_HOST":          "localhost",
		"SMTP_PORT":          "1025",
		"SMTP_TLS":           "none",
		"SMTP_USER":          "",
		"SMTP_PASSWORD":      "",
		"AUTH_BASE_URL":      "http://localhost:8080",
		"OIDC_ISSUER_URL":    "https://id.example.com",
		"OIDC_CLIENT_ID":     "my-car",
		"OIDC_CLIENT_SECRET": "secret",
	}
	for key, value := range values {
		t.Setenv(key, value)
	}
}

func TestLoad_AllPresent(t *testing.T) {
	setAllEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	want := &Config{
		Port:             "8080",
		DatabaseURL:      "postgres://mycars:mycars@localhost:5432/mycars?sslmode=disable",
		SMTPFrom:         "admin@example.com",
		SMTPHost:         "localhost",
		SMTPPort:         "1025",
		SMTPTLS:          "none",
		SMTPUser:         "",
		SMTPPassword:     "",
		AuthBaseURL:      "http://localhost:8080",
		OIDCIssuerURL:    "https://id.example.com",
		OIDCClientID:     "my-car",
		OIDCClientSecret: "secret",
		RateLimitPerIP:   true,
		LogLevel:         slog.LevelInfo,
		LogRequests:      RequestLogAll,
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Fatalf("Load() = %+v, want %+v", *cfg, *want)
	}
}

func TestLoad_MissingVariable(t *testing.T) {
	for _, missingKey := range requiredEnvKeys {
		t.Run(missingKey, func(t *testing.T) {
			setAllEnv(t)
			t.Setenv(missingKey, "")

			_, err := Load()
			if err == nil {
				t.Fatalf("Load() with %s unset: expected error, got nil", missingKey)
			}
			if !strings.Contains(err.Error(), missingKey) {
				t.Fatalf("Load() error %q does not mention missing key %s", err.Error(), missingKey)
			}
		})
	}
}

func TestLoad_OptionalSMTPAuthentication(t *testing.T) {
	setAllEnv(t)
	t.Setenv("SMTP_USER", "mailer")
	t.Setenv("SMTP_PASSWORD", "secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if cfg.SMTPUser != "mailer" || cfg.SMTPPassword != "secret" {
		t.Fatalf("Load() SMTP credentials = %q, %q; want configured values", cfg.SMTPUser, cfg.SMTPPassword)
	}
}

func TestLoad_OptionalOIDCConfiguration(t *testing.T) {
	setAllEnv(t)
	for _, key := range oidcEnvKeys {
		t.Setenv(key, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if cfg.OIDCEnabled() {
		t.Fatal("OIDCEnabled() = true with all OIDC values absent")
	}
}

func TestLoad_CompleteOIDCConfigurationIsEnabled(t *testing.T) {
	setAllEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if !cfg.OIDCEnabled() {
		t.Fatal("OIDCEnabled() = false with complete OIDC configuration")
	}
}

func TestLoad_RejectsPartialOIDCConfiguration(t *testing.T) {
	for mask := 1; mask < 1<<len(oidcEnvKeys)-1; mask++ {
		t.Run(fmt.Sprintf("combination_%d", mask), func(t *testing.T) {
			setAllEnv(t)
			for index, key := range oidcEnvKeys {
				if mask&(1<<index) == 0 {
					t.Setenv(key, "")
				}
			}

			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), "must be configured together") {
				t.Fatalf("Load() error = %v", err)
			}
		})
	}
}

func TestLoad_RejectsInvalidAuthenticationConfiguration(t *testing.T) {
	for key, value := range map[string]string{"AUTH_BASE_URL": "relative", "OIDC_ISSUER_URL": "javascript:bad", "SMTP_TLS": "sometimes"} {
		t.Run(key, func(t *testing.T) {
			setAllEnv(t)
			t.Setenv(key, value)
			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("Load() error=%v", err)
			}
		})
	}
}

func TestLoad_RejectsAuthBaseURLWithPath(t *testing.T) {
	setAllEnv(t)
	t.Setenv("AUTH_BASE_URL", "https://cars.example/subpath")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "AUTH_BASE_URL") {
		t.Fatalf("Load() error=%v", err)
	}
}

func TestLoad_TrustedProxies(t *testing.T) {
	for raw, want := range map[string][]netip.Prefix{
		"":                                nil,
		"172.30.0.2":                      {netip.MustParsePrefix("172.30.0.2/32")},
		" 172.30.0.0/24 , 10.1.2.3/8 ":    {netip.MustParsePrefix("172.30.0.0/24"), netip.MustParsePrefix("10.0.0.0/8")},
		"2001:db8::1, 2001:db8:abcd::/48": {netip.MustParsePrefix("2001:db8::1/128"), netip.MustParsePrefix("2001:db8:abcd::/48")},
		"::ffff:192.0.2.7":                {netip.MustParsePrefix("192.0.2.7/32")},
	} {
		setAllEnv(t)
		t.Setenv("TRUSTED_PROXIES", raw)
		cfg, err := Load()
		if err != nil {
			t.Fatalf("%q: %v", raw, err)
		}
		if !reflect.DeepEqual(cfg.TrustedProxies, want) {
			t.Errorf("%q: TrustedProxies = %v, want %v", raw, cfg.TrustedProxies, want)
		}
	}
	for _, raw := range []string{"proxy.example", "172.30.0.0/33", "172.30.0.2, nope"} {
		setAllEnv(t)
		t.Setenv("TRUSTED_PROXIES", raw)
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), "TRUSTED_PROXIES") {
			t.Errorf("%q: error = %v, want one naming TRUSTED_PROXIES", raw, err)
		}
	}
}

func TestLoad_RateLimitPerIP(t *testing.T) {
	for raw, want := range map[string]bool{"": true, "true": true, "false": false} {
		setAllEnv(t)
		t.Setenv("RATE_LIMIT_PER_IP", raw)
		cfg, err := Load()
		if err != nil || cfg.RateLimitPerIP != want {
			t.Errorf("%q: RateLimitPerIP = %v, err %v; want %v", raw, cfg != nil && cfg.RateLimitPerIP, err, want)
		}
	}
	for _, raw := range []string{"yes", "0", "FALSE"} {
		setAllEnv(t)
		t.Setenv("RATE_LIMIT_PER_IP", raw)
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), "RATE_LIMIT_PER_IP") {
			t.Errorf("%q: error = %v, want one naming RATE_LIMIT_PER_IP", raw, err)
		}
	}
}

func TestLoad_AnalyticsSnippet(t *testing.T) {
	snippet := `<script defer src="https://stats.example/script.js" data-website-id="abc"></script>`
	for raw, want := range map[string]string{"": "", "   \n\t": "", snippet: snippet} {
		setAllEnv(t)
		t.Setenv("ANALYTICS_SNIPPET", raw)
		cfg, err := Load()
		if err != nil || cfg.AnalyticsSnippet != want {
			t.Errorf("%q: Load() = %+v, err %v; want AnalyticsSnippet %q", raw, cfg, err, want)
		}
	}
}

func TestLoad_LogLevel(t *testing.T) {
	for raw, want := range map[string]slog.Level{"": slog.LevelInfo, "debug": slog.LevelDebug, "info": slog.LevelInfo, "warn": slog.LevelWarn, "error": slog.LevelError} {
		setAllEnv(t)
		t.Setenv("LOG_LEVEL", raw)
		cfg, err := Load()
		if err != nil || cfg.LogLevel != want {
			t.Errorf("%q: Load() = %+v, err %v; want LogLevel %s", raw, cfg, err, want)
		}
	}
	for _, raw := range []string{"verbose", "INFO", "warning"} {
		setAllEnv(t)
		t.Setenv("LOG_LEVEL", raw)
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), "LOG_LEVEL") {
			t.Errorf("%q: error = %v, want one naming LOG_LEVEL", raw, err)
		}
	}
}

func TestLoad_LogRequests(t *testing.T) {
	for raw, want := range map[string]RequestLogMode{"": RequestLogAll, "all": RequestLogAll, "errors": RequestLogErrors, "off": RequestLogOff} {
		setAllEnv(t)
		t.Setenv("LOG_REQUESTS", raw)
		cfg, err := Load()
		if err != nil || cfg.LogRequests != want {
			t.Errorf("%q: Load() = %+v, err %v; want LogRequests %q", raw, cfg, err, want)
		}
	}
	for _, raw := range []string{"some", "ALL", "false"} {
		setAllEnv(t)
		t.Setenv("LOG_REQUESTS", raw)
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), "LOG_REQUESTS") {
			t.Errorf("%q: error = %v, want one naming LOG_REQUESTS", raw, err)
		}
	}
}
