// Package config loads the backend's runtime configuration from
// environment variables.
package config

import (
	"fmt"
	"log/slog"
	"net/netip"
	"net/url"
	"os"
	"strings"
)

// Config holds every environment variable the backend reads.
type Config struct {
	Port             string
	DatabaseURL      string
	SMTPFrom         string
	SMTPHost         string
	SMTPPort         string
	SMTPTLS          string
	SMTPUser         string
	SMTPPassword     string
	AuthBaseURL      string
	OIDCIssuerURL    string
	OIDCClientID     string
	OIDCClientSecret string
	// TrustedProxies lists the reverse proxies whose X-Forwarded-For header
	// is believed when resolving a client's address for rate limiting.
	TrustedProxies []netip.Prefix
	// RateLimitPerIP enables per-client-address throttling of the public
	// authentication endpoints.
	RateLimitPerIP bool
	// AnalyticsSnippet is raw operator-provided HTML inserted into the
	// frontend's index.html before </head>; empty when unset.
	AnalyticsSnippet string
	// LogLevel is the minimum level of all log output.
	LogLevel slog.Level
	// LogRequests selects which HTTP requests get an access-log entry.
	LogRequests RequestLogMode
}

// RequestLogMode selects which HTTP requests get an access-log entry.
type RequestLogMode string

const (
	// RequestLogAll logs every request.
	RequestLogAll RequestLogMode = "all"
	// RequestLogErrors logs only responses with status 400 or higher.
	RequestLogErrors RequestLogMode = "errors"
	// RequestLogOff disables the access log.
	RequestLogOff RequestLogMode = "off"
)

// EnvKeys lists every environment variable Load reads, in the order
// backend/.env.example documents them. config_test.go checks this list
// against .env.example's variable names so the two can't drift apart.
var EnvKeys = []string{
	"PORT",
	"DATABASE_URL",
	"SMTP_FROM",
	"SMTP_HOST",
	"SMTP_PORT",
	"SMTP_TLS",
	"SMTP_USER",
	"SMTP_PASSWORD",
	"AUTH_BASE_URL",
	"OIDC_ISSUER_URL",
	"OIDC_CLIENT_ID",
	"OIDC_CLIENT_SECRET",
	"TRUSTED_PROXIES",
	"RATE_LIMIT_PER_IP",
	"ANALYTICS_SNIPPET",
	"LOG_LEVEL",
	"LOG_REQUESTS",
}

var requiredEnvKeys = []string{
	"PORT",
	"DATABASE_URL",
	"SMTP_FROM",
	"SMTP_HOST",
	"SMTP_PORT",
	"SMTP_TLS",
	"AUTH_BASE_URL",
}

var oidcEnvKeys = []string{"OIDC_ISSUER_URL", "OIDC_CLIENT_ID", "OIDC_CLIENT_SECRET"}

// Load reads Config from the environment, returning an error naming every
// missing required variable.
func Load() (*Config, error) {
	values := make(map[string]string, len(EnvKeys))
	var missing []string
	for _, key := range requiredEnvKeys {
		v := os.Getenv(key)
		if v == "" {
			missing = append(missing, key)
		}
		values[key] = v
	}
	for _, key := range append([]string{"SMTP_USER", "SMTP_PASSWORD", "TRUSTED_PROXIES", "RATE_LIMIT_PER_IP", "ANALYTICS_SNIPPET", "LOG_LEVEL", "LOG_REQUESTS"}, oidcEnvKeys...) {
		values[key] = os.Getenv(key)
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variable(s): %s", strings.Join(missing, ", "))
	}
	configuredOIDCValues := 0
	for _, key := range oidcEnvKeys {
		if values[key] != "" {
			configuredOIDCValues++
		}
	}
	if configuredOIDCValues != 0 && configuredOIDCValues != len(oidcEnvKeys) {
		return nil, fmt.Errorf("OIDC_ISSUER_URL, OIDC_CLIENT_ID, and OIDC_CLIENT_SECRET must be configured together")
	}
	urlKeys := []string{"AUTH_BASE_URL"}
	if configuredOIDCValues == len(oidcEnvKeys) {
		urlKeys = append(urlKeys, "OIDC_ISSUER_URL")
	}
	for _, key := range urlKeys {
		u, err := url.Parse(values[key])
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return nil, fmt.Errorf("%s must be an absolute HTTP(S) URL", key)
		}
	}
	baseURL, _ := url.Parse(values["AUTH_BASE_URL"])
	if (baseURL.Path != "" && baseURL.Path != "/") || baseURL.RawQuery != "" || baseURL.Fragment != "" {
		return nil, fmt.Errorf("AUTH_BASE_URL must contain only an origin")
	}
	if values["SMTP_TLS"] != "none" && values["SMTP_TLS"] != "starttls" && values["SMTP_TLS"] != "tls" {
		return nil, fmt.Errorf("SMTP_TLS must be one of none, starttls, tls")
	}
	trustedProxies, err := parseTrustedProxies(values["TRUSTED_PROXIES"])
	if err != nil {
		return nil, err
	}
	rateLimitPerIP := true
	switch values["RATE_LIMIT_PER_IP"] {
	case "", "true":
	case "false":
		rateLimitPerIP = false
	default:
		return nil, fmt.Errorf("RATE_LIMIT_PER_IP must be true or false")
	}

	// An all-whitespace value is treated as unset so a stray space in a
	// compose file doesn't alter every served page.
	analyticsSnippet := values["ANALYTICS_SNIPPET"]
	if strings.TrimSpace(analyticsSnippet) == "" {
		analyticsSnippet = ""
	}

	var logLevel slog.Level
	switch values["LOG_LEVEL"] {
	case "", "info":
		logLevel = slog.LevelInfo
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		return nil, fmt.Errorf("LOG_LEVEL must be one of debug, info, warn, error")
	}
	logRequests := RequestLogMode(values["LOG_REQUESTS"])
	switch logRequests {
	case "":
		logRequests = RequestLogAll
	case RequestLogAll, RequestLogErrors, RequestLogOff:
	default:
		return nil, fmt.Errorf("LOG_REQUESTS must be one of all, errors, off")
	}

	return &Config{
		Port:             values["PORT"],
		DatabaseURL:      values["DATABASE_URL"],
		SMTPFrom:         values["SMTP_FROM"],
		SMTPHost:         values["SMTP_HOST"],
		SMTPPort:         values["SMTP_PORT"],
		SMTPTLS:          values["SMTP_TLS"],
		SMTPUser:         values["SMTP_USER"],
		SMTPPassword:     values["SMTP_PASSWORD"],
		AuthBaseURL:      values["AUTH_BASE_URL"],
		OIDCIssuerURL:    values["OIDC_ISSUER_URL"],
		OIDCClientID:     values["OIDC_CLIENT_ID"],
		OIDCClientSecret: values["OIDC_CLIENT_SECRET"],
		TrustedProxies:   trustedProxies,
		RateLimitPerIP:   rateLimitPerIP,
		AnalyticsSnippet: analyticsSnippet,
		LogLevel:         logLevel,
		LogRequests:      logRequests,
	}, nil
}

// parseTrustedProxies reads a comma-separated list of IP addresses and CIDR
// ranges; a bare address trusts exactly that host.
func parseTrustedProxies(raw string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if prefix, err := netip.ParsePrefix(entry); err == nil {
			out = append(out, prefix.Masked())
			continue
		}
		addr, err := netip.ParseAddr(entry)
		if err != nil {
			return nil, fmt.Errorf("TRUSTED_PROXIES entry %q is neither an IP address nor a CIDR range", entry)
		}
		addr = addr.Unmap()
		out = append(out, netip.PrefixFrom(addr, addr.BitLen()))
	}
	return out, nil
}

// OIDCEnabled reports whether the complete optional OIDC configuration is set.
func (c *Config) OIDCEnabled() bool {
	return c.OIDCIssuerURL != "" && c.OIDCClientID != "" && c.OIDCClientSecret != ""
}
