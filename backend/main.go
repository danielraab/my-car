package main

import (
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"

	"at.draab/my-car/internal/attachments"
	"at.draab/my-car/internal/auth"
	"at.draab/my-car/internal/cars"
	"at.draab/my-car/internal/config"
	"at.draab/my-car/internal/db"
	"at.draab/my-car/internal/healthcheck"
	"at.draab/my-car/internal/httpserver"
	"at.draab/my-car/internal/legacyimport"
	"at.draab/my-car/internal/profile"
	"at.draab/my-car/internal/refuels"
	"at.draab/my-car/internal/repairs"
	"at.draab/my-car/internal/seed"
	"at.draab/my-car/internal/stats"
	"at.draab/my-car/internal/tickets"
)

//go:embed openapi.yaml
var openapiDoc []byte

//go:embed all:static/out
var staticFS embed.FS

func main() {
	if isHealthcheckCommand(os.Args) {
		runHealthcheck()
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "seed" {
		if err := runSeed(os.Args[2:]); err != nil {
			fatal("seed", err)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "import-legacy" {
		os.Exit(runImportLegacy(os.Args[2:], os.Stdout, os.Stderr))
	}
	runServer()
}

// importLegacyArgs parses `import-legacy <dump.sql> [--dry-run]`.
func importLegacyArgs(args []string) (path string, dryRun bool, err error) {
	for _, arg := range args {
		switch {
		case arg == "--dry-run":
			dryRun = true
		case strings.HasPrefix(arg, "-") || path != "":
			return "", false, fmt.Errorf("usage: backend import-legacy <dump.sql> [--dry-run] (imports once, into a fresh database)")
		default:
			path = arg
		}
	}
	if path == "" {
		return "", false, fmt.Errorf("usage: backend import-legacy <dump.sql> [--dry-run] (imports once, into a fresh database)")
	}
	return path, dryRun, nil
}

// runImportLegacy validates the dump and, unless dryRun, writes it. Problems
// go to stdout, one per line and nothing else, so a dry run lists only them;
// it returns the process exit code.
func runImportLegacy(args []string, stdout, stderr io.Writer) int {
	path, dryRun, err := importLegacyArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	dump, err := readLegacyDump(path)
	if err != nil {
		fmt.Fprintf(stdout, "dump: %v\n", err)
		return 1
	}
	plan, problems := legacyimport.Map(dump)
	for _, p := range problems {
		fmt.Fprintln(stdout, p)
	}
	if len(problems) > 0 {
		return 1
	}
	if dryRun {
		fmt.Fprintln(stdout, "no problems found")
		return 0
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		fmt.Fprintln(stderr, "import-legacy: DATABASE_URL is required")
		return 1
	}
	if err := db.RunMigrations(databaseURL); err != nil {
		fmt.Fprintf(stderr, "import-legacy: run migrations: %v\n", err)
		return 1
	}
	ctx := context.Background()
	pool, err := db.NewPool(ctx, databaseURL)
	if err != nil {
		fmt.Fprintf(stderr, "import-legacy: %v\n", err)
		return 1
	}
	defer pool.Close()
	counts, err := legacyimport.Write(ctx, pool, plan)
	if err != nil {
		fmt.Fprintf(stderr, "import-legacy: %v; nothing imported\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "imported %d accounts, %d cars, %d refuels, %d repairs, %d tickets\n",
		counts.Accounts, counts.Cars, counts.Refuels, counts.Repairs, counts.Tickets)
	return 0
}

func readLegacyDump(path string) (legacyimport.Dump, error) {
	f, err := os.Open(path)
	if err != nil {
		return legacyimport.Dump{}, err
	}
	defer f.Close()
	return legacyimport.Parse(f)
}

func seedEmails(args []string) ([]string, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: backend seed <email> [email ...] (replaces all application data)")
	}
	var emails []string
	seen := make(map[string]bool)
	for _, arg := range args {
		email := strings.ToLower(strings.TrimSpace(arg))
		parsed, err := mail.ParseAddress(email)
		if err != nil || parsed.Address != email || parsed.Name != "" || strings.ContainsAny(email, " \t\n") {
			return nil, fmt.Errorf("invalid email address: %q", arg)
		}
		if !seen[email] {
			emails = append(emails, email)
			seen[email] = true
		}
	}
	return emails, nil
}

func runSeed(args []string) error {
	emails, err := seedEmails(args)
	if err != nil {
		return err
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if err := db.RunMigrations(databaseURL); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	ctx := context.Background()
	pool, err := db.NewPool(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := seed.Run(ctx, pool, emails, time.Now()); err != nil {
		return err
	}
	slog.Info("seeded accounts", "accounts", len(emails), "cars_each", 4, "refuels_each", 500, "repairs_each", 75, "tickets_each", 90)
	return nil
}

// isHealthcheckCommand reports whether args (as os.Args) select the
// healthcheck subcommand rather than the default server mode.
func isHealthcheckCommand(args []string) bool {
	return len(args) > 1 && args[1] == "healthcheck"
}

// runHealthcheck is the container HEALTHCHECK's entry point: probe our own
// /api/healthz and report success or failure via the process exit code,
// since the distroless runtime image has no curl.
func runHealthcheck() {
	cfg, err := config.Load()
	if err != nil {
		fatal("healthcheck: load config", err)
	}

	if err := healthcheck.Probe(cfg.Port); err != nil {
		slog.Error("healthcheck", "error", err)
		os.Exit(1)
	}
}

func runServer() {
	cfg, err := config.Load()
	if err != nil {
		fatal("load config", err)
	}
	// Plain key=value text; output of the standard log package (such as
	// net/http's own errors) is routed through this handler too.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: cfg.LogLevel})))

	if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
		fatal("run migrations", err)
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		fatal("connect to database", err)
	}
	defer pool.Close()

	staticOut, err := fs.Sub(staticFS, "static/out")
	if err != nil {
		fatal("prepare embedded static assets", err)
	}
	staticOut, err = httpserver.WithAnalyticsSnippet(staticOut, cfg.AnalyticsSnippet)
	if err != nil {
		fatal("prepare analytics snippet", err)
	}
	if cfg.AnalyticsSnippet != "" {
		slog.Info("analytics snippet configured")
	}

	store := auth.NewStore(pool)
	if err := store.Prune(ctx, time.Now()); err != nil {
		slog.Error("prune expired authentication state", "error", err)
	}
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for now := range ticker.C {
			if err := store.Prune(ctx, now); err != nil {
				slog.Error("prune expired authentication state", "error", err)
			}
		}
	}()
	var oidcClient auth.OIDCProvider
	if cfg.OIDCEnabled() {
		oidcClient, err = auth.NewOIDCClient(ctx, cfg.OIDCIssuerURL, cfg.OIDCClientID, cfg.OIDCClientSecret, cfg.AuthBaseURL+"/api/v1/auth/oidc/callback")
		if err != nil {
			fatal("configure OIDC", err)
		}
	}
	mailer := auth.NewSMTPMailer(cfg.SMTPFrom, cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPTLS, cfg.SMTPUser, cfg.SMTPPassword)
	authService := auth.NewService(store, mailer, oidcClient, cfg.AuthBaseURL)
	authService.ConfigureRateLimiting(cfg.TrustedProxies, cfg.RateLimitPerIP)
	blobs, err := newBlobStore(ctx, cfg)
	if err != nil {
		fatal("configure file storage", err)
	}
	attachmentStore := attachments.NewStore(pool)
	cleaner := attachments.NewCleaner(attachmentStore, blobs)
	go cleaner.Run(ctx, time.Minute)
	attachmentsHandler := attachments.NewHandler(attachmentStore, blobs, cfg.MaxUploadBytes, cleaner.Kick)
	profileHandler := profile.NewHandler(profile.NewStore(pool))
	carsHandler := cars.NewHandler(cars.NewStore(pool))
	repairsHandler := repairs.NewHandler(repairs.NewStore(pool))
	refuelsHandler := refuels.NewHandler(refuels.NewStore(pool))
	ticketsHandler := tickets.NewHandler(tickets.NewStore(pool))
	statsHandler := stats.NewHandler(stats.NewStore(pool))
	mux := httpserver.NewMux(pool, openapiDoc, staticOut, authService.RegisterRoutes, func(mux *http.ServeMux) {
		profileHandler.RegisterRoutes(mux, authService.RequireSession)
		carsHandler.RegisterRoutes(mux, authService.RequireSession)
		repairsHandler.RegisterRoutes(mux, authService.RequireSession)
		refuelsHandler.RegisterRoutes(mux, authService.RequireSession)
		ticketsHandler.RegisterRoutes(mux, authService.RequireSession)
		statsHandler.RegisterRoutes(mux, authService.RequireSession)
		attachmentsHandler.RegisterRoutes(mux, authService.RequireSession)
	})

	slog.Info("listening", "port", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, httpserver.Wrap(mux, cfg.LogRequests)); err != nil {
		fatal("serve", err)
	}
}

// newBlobStore opens the configured attachment storage and checks that it
// is usable, so a misconfiguration stops startup instead of failing the
// first upload. Errors name the variables to fix.
func newBlobStore(ctx context.Context, cfg *config.Config) (attachments.BlobStore, error) {
	if cfg.FileStorage == config.FileStorageS3 {
		store, err := attachments.NewS3Store(cfg.S3)
		if err != nil {
			return nil, fmt.Errorf("S3_ENDPOINT, S3_ACCESS_KEY_ID, S3_SECRET_ACCESS_KEY: %w", err)
		}
		checkCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		if err := store.Check(checkCtx); err != nil {
			return nil, fmt.Errorf("S3 bucket is not usable (check S3_ENDPOINT, S3_BUCKET, S3_REGION and the credentials): %w", err)
		}
		slog.Info("file storage configured", "backend", "s3", "bucket", cfg.S3.Bucket)
		return store, nil
	}
	store, err := attachments.NewLocalStore(cfg.FileStoragePath)
	if err != nil {
		return nil, fmt.Errorf("FILE_STORAGE_PATH %q cannot be created: %w", cfg.FileStoragePath, err)
	}
	if err := store.Check(ctx); err != nil {
		return nil, fmt.Errorf("FILE_STORAGE_PATH %q is not writable: %w", cfg.FileStoragePath, err)
	}
	slog.Info("file storage configured", "backend", "local", "path", cfg.FileStoragePath)
	return store, nil
}

// fatal logs err at error level and exits, like log.Fatalf did.
func fatal(msg string, err error) {
	slog.Error(msg, "error", err)
	os.Exit(1)
}
