// Package config loads the backend's runtime configuration from
// environment variables.
package config

import (
	"fmt"
	"log/slog"
	"net/netip"
	"net/url"
	"os"
	"strconv"
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
	// FileStorage selects where attachment contents are stored.
	FileStorage FileStorageBackend
	// FileStoragePath is the directory of local storage.
	FileStoragePath string
	// S3 holds the S3 storage settings; zero unless FileStorage is s3.
	S3 S3Config
	// MaxUploadBytes is the maximum size of a single attachment.
	MaxUploadBytes int64
}

// FileStorageBackend selects where attachment contents are stored.
type FileStorageBackend string

const (
	// FileStorageLocal stores attachments below FileStoragePath.
	FileStorageLocal FileStorageBackend = "local"
	// FileStorageS3 stores attachments in an S3-compatible bucket.
	FileStorageS3 FileStorageBackend = "s3"
)

// DefaultFileStoragePath is the local storage directory when
// FILE_STORAGE_PATH is unset, relative to the working directory.
const DefaultFileStoragePath = "./data/files"

// DefaultMaxUploadBytes is the attachment size limit when MAX_UPLOAD_BYTES is
// unset (10 MB).
const DefaultMaxUploadBytes int64 = 10 << 20

// S3Config holds the settings of S3-compatible attachment storage.
type S3Config struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool
	Prefix          string
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
	"FILE_STORAGE",
	"FILE_STORAGE_PATH",
	"S3_ENDPOINT",
	"S3_REGION",
	"S3_BUCKET",
	"S3_ACCESS_KEY_ID",
	"S3_SECRET_ACCESS_KEY",
	"S3_USE_PATH_STYLE",
	"S3_PREFIX",
	"MAX_UPLOAD_BYTES",
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

var fileStorageEnvKeys = []string{"FILE_STORAGE", "FILE_STORAGE_PATH", "S3_ENDPOINT", "S3_REGION", "S3_BUCKET", "S3_ACCESS_KEY_ID", "S3_SECRET_ACCESS_KEY", "S3_USE_PATH_STYLE", "S3_PREFIX", "MAX_UPLOAD_BYTES"}

var requiredS3EnvKeys = []string{"S3_ENDPOINT", "S3_BUCKET", "S3_ACCESS_KEY_ID", "S3_SECRET_ACCESS_KEY"}

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
	for _, key := range append([]string{"SMTP_USER", "SMTP_PASSWORD", "TRUSTED_PROXIES", "RATE_LIMIT_PER_IP", "ANALYTICS_SNIPPET", "LOG_LEVEL", "LOG_REQUESTS"}, append(oidcEnvKeys, fileStorageEnvKeys...)...) {
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

	fileStorage, fileStoragePath, s3, err := parseFileStorage(values)
	if err != nil {
		return nil, err
	}
	maxUploadBytes := DefaultMaxUploadBytes
	if raw := values["MAX_UPLOAD_BYTES"]; raw != "" {
		maxUploadBytes, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || maxUploadBytes <= 0 {
			return nil, fmt.Errorf("MAX_UPLOAD_BYTES must be a positive number of bytes")
		}
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
		FileStorage:      fileStorage,
		FileStoragePath:  fileStoragePath,
		S3:               s3,
		MaxUploadBytes:   maxUploadBytes,
	}, nil
}

// parseFileStorage reads the attachment storage selection. The S3 variables
// are only read, and required, when FILE_STORAGE is s3.
func parseFileStorage(values map[string]string) (FileStorageBackend, string, S3Config, error) {
	backend := FileStorageBackend(values["FILE_STORAGE"])
	switch backend {
	case "", FileStorageLocal:
		path := values["FILE_STORAGE_PATH"]
		if path == "" {
			path = DefaultFileStoragePath
		}
		return FileStorageLocal, path, S3Config{}, nil
	case FileStorageS3:
	default:
		return "", "", S3Config{}, fmt.Errorf("FILE_STORAGE must be one of local, s3")
	}
	var missing []string
	for _, key := range requiredS3EnvKeys {
		if values[key] == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return "", "", S3Config{}, fmt.Errorf("FILE_STORAGE=s3 requires environment variable(s): %s", strings.Join(missing, ", "))
	}
	endpoint, err := url.Parse(values["S3_ENDPOINT"])
	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") {
		return "", "", S3Config{}, fmt.Errorf("S3_ENDPOINT must be an absolute HTTP(S) URL")
	}
	usePathStyle := false
	switch values["S3_USE_PATH_STYLE"] {
	case "", "false":
	case "true":
		usePathStyle = true
	default:
		return "", "", S3Config{}, fmt.Errorf("S3_USE_PATH_STYLE must be true or false")
	}
	return FileStorageS3, "", S3Config{
		Endpoint:        values["S3_ENDPOINT"],
		Region:          values["S3_REGION"],
		Bucket:          values["S3_BUCKET"],
		AccessKeyID:     values["S3_ACCESS_KEY_ID"],
		SecretAccessKey: values["S3_SECRET_ACCESS_KEY"],
		UsePathStyle:    usePathStyle,
		Prefix:          values["S3_PREFIX"],
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
