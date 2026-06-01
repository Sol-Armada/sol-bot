package main

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/sol-armada/sol-bot/activity"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/bot"
	"github.com/sol-armada/sol-bot/config"
	"github.com/sol-armada/sol-bot/database"
	"github.com/sol-armada/sol-bot/database/migrations"
	"github.com/sol-armada/sol-bot/database/postgresql"
	"github.com/sol-armada/sol-bot/giveaway"
	"github.com/sol-armada/sol-bot/health"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/raffles"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/systemd"
	"github.com/sol-armada/sol-bot/tokens"
)

var (
	environment string = ""
	version     string = "dev"
	hash        string = "unknown"
	logger      *slog.Logger
)

// Config holds all application configuration
type Config struct {
	Environment string
	Debug       bool
	Database    database.Config
}

func init() {
	fmt.Println("Starting sol-bot initialization...")

	cfg := loadConfig()
	logger = setupLogger(cfg)

	logger.Info("sol-bot starting up",
		"environment", cfg.Environment,
		"debug", cfg.Debug,
		"version", version,
		"commit", hash)

	logger.Info("configuration loaded successfully")

	if err := initializeServices(cfg); err != nil {
		logger.Error("failed to initialize services", "error", err)
		os.Exit(1)
	}

	logger.Info("health monitor starting")
	go health.Monitor()
}

// loadConfig initializes and loads application configuration
func loadConfig() *Config {
	appEnv := resolveEnvironment()
	fmt.Printf("Loading configuration for environment: %s\n", appEnv)

	configureSettingsEnvironment()

	configFile := resolveConfigFile()
	if configFile == "" {
		fmt.Printf("No configuration file found, relying on environment variables and code defaults\n")
	} else {
		settings.SetConfigFile(configFile)
		fmt.Printf("Attempting to read configuration file: %s\n", configFile)
		if err := settings.ReadInConfig(); err != nil {
			fmt.Printf("could not parse configuration: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Configuration loaded successfully\n")
	}

	return &Config{
		Environment: appEnv,
		Debug:       settings.GetBool("LOG.DEBUG"),
		Database: database.Config{
			Postgres: database.PostgresConfig{
				Host:           settings.GetStringWithDefault("postgres.host", "localhost"),
				Port:           settings.GetIntWithDefault("postgres.port", 5432),
				Username:       settings.GetStringWithDefault("postgres.username", ""),
				Password:       settings.GetStringWithDefault("postgres.password", ""),
				Database:       settings.GetStringWithDefault("postgres.database", "org"),
				SSLMode:        settings.GetStringWithDefault("postgres.ssl_mode", "disable"),
				MaxConns:       int32(settings.GetIntWithDefault("postgres.max_conns", 10)),
				MinConns:       int32(settings.GetIntWithDefault("postgres.min_conns", 1)),
				ConnectTimeout: parseDurationWithDefault("postgres.connect_timeout", 5*time.Second),
			},
		},
	}
}

func resolveEnvironment() string {
	if appEnv := strings.TrimSpace(os.Getenv("APP_ENV")); appEnv != "" {
		return appEnv
	}

	if appEnv := strings.TrimSpace(environment); appEnv != "" {
		return appEnv
	}

	return "local"
}

func resolveConfigFile() string {
	candidates := []string{}

	if configuredPath := strings.TrimSpace(os.Getenv("SOLBOT_CONFIG_FILE")); configuredPath != "" {
		candidates = append(candidates, configuredPath)
	}

	candidates = append(candidates,
		"/etc/solbot/config.toml",
		"./settings.toml",
		"../settings.toml",
	)

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func configureSettingsEnvironment() {
	settings.SetEnvPrefix("SOLBOT")
	settings.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	settings.AutomaticEnv()

	legacyEnvBindings := map[string][]string{
		"postgres.host":            {"POSTGRES_HOST"},
		"postgres.port":            {"POSTGRES_PORT"},
		"postgres.username":        {"POSTGRES_USERNAME"},
		"postgres.password":        {"POSTGRES_PASSWORD"},
		"postgres.database":        {"POSTGRES_DATABASE"},
		"postgres.ssl_mode":        {"POSTGRES_SSL_MODE"},
		"postgres.max_conns":       {"POSTGRES_MAX_CONNS"},
		"postgres.min_conns":       {"POSTGRES_MIN_CONNS"},
		"postgres.connect_timeout": {"POSTGRES_CONNECT_TIMEOUT"},
	}

	for key, envNames := range legacyEnvBindings {
		if err := settings.BindEnv(key, envNames...); err != nil {
			fmt.Printf("warning: failed to bind env for %s: %v\n", key, err)
		}
	}
}

func parseDurationWithDefault(key string, fallback time.Duration) time.Duration {
	raw := settings.GetStringWithDefault(key, fallback.String())
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}

// setupLogger creates and configures the application logger
func setupLogger(cfg *Config) *slog.Logger {
	fmt.Printf("Setting up logger - Debug: %v\n", cfg.Debug)

	opts := &slog.HandlerOptions{
		AddSource: true,
	}

	if cfg.Debug {
		opts.Level = slog.LevelDebug
		fmt.Printf("Debug logging enabled\n")
	}

	var handler slog.Handler
	if settings.GetBool("LOG_HUMAN") {
		handler = slog.NewTextHandler(os.Stdout, opts)
		fmt.Printf("Using human-readable log format\n")
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
		fmt.Printf("Using JSON log format\n")
	}

	log := slog.New(handler)
	slog.SetDefault(log)

	if cfg.Debug {
		log.Debug("debug mode enabled")
	}

	fmt.Printf("Logger setup completed successfully\n")
	return log
}

// initializeServices sets up database connection and initializes all services
func initializeServices(cfg *Config) error {
	ctx := context.Background()

	// Initialize database connection
	logger.Info("initializing database connection")

	pgClient, err := postgresql.New(ctx, cfg.Database.Postgres)
	if err != nil {
		return fmt.Errorf("failed to initialize postgres client: %w", err)
	}
	logger.Info("database connection established successfully")

	// Run migrations
	logger.Info("running schema migrations")
	runner := migrations.New(pgClient.Pool, ctx)
	if err := runner.ApplyAll("database/migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	migrationsApplied, err := runner.GetAppliedMigrations()
	if err == nil {
		logger.Info("schema migrations completed", "count", len(migrationsApplied))
	}

	// Initialize PostgreSQL-migrated services only.
	services := map[string]func() error{
		"members":    members.Setup,
		"attendance": attendance.Setup,
		"tokens":     tokens.Setup,
		"activity":   activity.Setup,
		"giveaway":   giveaway.Setup,
		"raffles":    raffles.Setup,
		"config":     config.Setup,
	}

	logger.Info("initializing services", "count", len(services))
	for name, setup := range services {
		logger.Info("setting up service", "service", name)
		if err := setup(); err != nil {
			return fmt.Errorf("failed to setup %s service: %w", name, err)
		}
		logger.Info("service setup completed", "service", name)
	}

	logger.Info("all services initialized successfully")
	return nil
}

// Application represents the main application structure
type Application struct {
	cfg       *Config
	bot       *bot.Bot
	scheduler gocron.Scheduler
	logger    *slog.Logger

	// Channel for graceful shutdown
	stopCh chan struct{}
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			if logger != nil {
				logger.Error("panic in main", "panic", r)
			} else {
				fmt.Printf("panic in main: %v\n", r)
			}
		}
	}()

	logger.Info("main function starting")
	cfg := loadConfig()

	logger.Info("creating application instance")
	app := &Application{
		cfg:    cfg,
		logger: logger,
		stopCh: make(chan struct{}),
	}

	defer func() {
		logger.Info("shutting down application")
		app.shutdown()
		logger.Info("application shutdown completed")
	}()

	logger.Info("starting application")
	if err := app.start(); err != nil {
		logger.Error("failed to start application", "error", err)
		os.Exit(1)
	}

	logger.Info("application started successfully, waiting for shutdown signal")
	// Wait for shutdown signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	sig := <-c
	logger.Info("received shutdown signal", "signal", sig.String())
}

// start initializes and starts all application components
func (app *Application) start() error {
	logger.Info("starting application components")

	logger.Info("systemd integration enabled, setting startup status")
	if err := systemd.Status("Starting up..."); err != nil {
		logger.Warn("failed to set systemd status", "error", err)
	}

	// Initialize bot with exponential backoff
	logger.Info("initializing Discord bot")
	if err := app.initializeBotWithBackoff(); err != nil {
		logger.Error("failed to initialize bot after all retries", "error", err)
		return fmt.Errorf("failed to initialize bot after retries: %w", err)
	}
	logger.Info("Discord bot initialized successfully")

	// Initialize scheduler
	logger.Info("initializing job scheduler")
	if err := app.initializeScheduler(); err != nil {
		logger.Error("failed to initialize scheduler", "error", err)
		return fmt.Errorf("failed to initialize scheduler: %w", err)
	}
	logger.Info("job scheduler initialized successfully")

	logger.Info("notifying systemd that application is ready")
	if err := systemd.Ready(); err != nil {
		logger.Warn("failed to notify systemd ready", "error", err)
	}
	if err := systemd.Status("Bot ready and serving"); err != nil {
		logger.Warn("failed to set systemd status", "error", err)
	}
	logger.Info("systemd notifications sent successfully")

	logger.Info("application startup completed successfully")
	return nil
}

// initializeBot creates and sets up the Discord bot
func (app *Application) initializeBot() error {
	logger.Info("creating new bot instance")
	var err error
	app.bot, err = bot.New(version)
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}
	logger.Info("bot instance created successfully")

	if err := systemd.Status("Setting up bot..."); err != nil {
		logger.Warn("failed to set systemd status", "error", err)
	}

	logger.Info("setting up bot configuration and handlers")
	if err := app.bot.Setup(); err != nil {
		logger.Error("failed to setup bot configuration", "error", err)
		return fmt.Errorf("failed to setup bot: %w", err)
	}
	logger.Info("bot setup completed successfully")

	return nil
}

// initializeBotWithBackoff initializes the bot with exponential backoff retry logic
func (app *Application) initializeBotWithBackoff() error {
	maxAttempts := 10
	baseDelay := 30 * time.Second
	maxDelay := 600 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		logger.Info("attempting to initialize bot", "attempt", attempt, "max_attempts", maxAttempts)

		status := fmt.Sprintf("Initializing bot (attempt %d/%d)...", attempt, maxAttempts)
		if err := systemd.Status(status); err != nil {
			logger.Warn("failed to set systemd status", "error", err)
		}

		err := app.initializeBot()
		if err == nil {
			logger.Info("bot initialized successfully", "attempt", attempt)
			return nil
		}

		logger.Error("bot initialization failed", "attempt", attempt, "error", err)

		// Don't wait after the last attempt
		if attempt == maxAttempts {
			break
		}

		// Calculate exponential backoff delay: 2^(attempt-1) * baseDelay, capped at maxDelay
		delay := min(time.Duration(1<<(attempt-1))*baseDelay, maxDelay)

		logger.Info("retrying bot initialization", "delay", delay, "next_attempt", attempt+1)

		status = fmt.Sprintf("Initialization failed, retrying in %v...", delay)
		if err := systemd.Status(status); err != nil {
			logger.Warn("failed to set systemd status", "error", err)
		}

		time.Sleep(delay)
	}

	return fmt.Errorf("failed to initialize bot after %d attempts", maxAttempts)
}

// initializeScheduler creates and configures the job scheduler
func (app *Application) initializeScheduler() error {
	logger.Info("creating new job scheduler")
	var err error
	app.scheduler, err = gocron.NewScheduler()
	if err != nil {
		logger.Error("failed to create scheduler", "error", err)
		return fmt.Errorf("failed to create scheduler: %w", err)
	}

	logger.Info("starting job scheduler")
	app.scheduler.Start()

	// Schedule member monitoring if enabled
	logger.Info("scheduling member monitor job")
	if err := app.scheduleMemberMonitor(); err != nil {
		logger.Error("failed to schedule member monitor", "error", err)
		return fmt.Errorf("failed to schedule member monitor: %w", err)
	}
	logger.Info("member monitor job scheduled successfully")

	// Schedule status message updates
	logger.Info("scheduling status update job")
	if err := app.scheduleStatusUpdates(); err != nil {
		logger.Error("failed to schedule status updates", "error", err)
		return fmt.Errorf("failed to schedule status updates: %w", err)
	}
	logger.Info("status update job scheduled successfully")

	logger.Info("scheduling systemd watchdog job")
	if err := app.scheduleSystemdWatchdog(); err != nil {
		logger.Error("failed to schedule systemd watchdog", "error", err)
		return fmt.Errorf("failed to schedule systemd watchdog: %w", err)
	}
	logger.Info("systemd watchdog job scheduled successfully")

	logger.Info("job scheduler initialization completed")
	return nil
}

// createMonitorLogger creates a dedicated logger for monitoring tasks
func (app *Application) createMonitorLogger() *slog.Logger {
	opts := &slog.HandlerOptions{
		AddSource: true,
	}

	if app.cfg.Debug {
		opts.Level = slog.LevelDebug
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

// scheduleMemberMonitor sets up the member monitoring job
func (app *Application) scheduleMemberMonitor() error {
	monitorLogger := app.createMonitorLogger()

	logger.Info("scheduling member monitor")
	j, err := app.scheduler.NewJob(
		gocron.CronJob("*/30 * * * *", false),
		gocron.NewTask(func(ctx context.Context) error {
			return bot.MemberMonitor(ctx, monitorLogger)
		}),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
		gocron.WithEventListeners(
			gocron.BeforeJobRuns(func(jobID uuid.UUID, jobName string) {
				monitorLogger.Info("starting member monitor", "job_id", jobID, "job_name", jobName)
			}),
			gocron.AfterJobRunsWithError(func(jobID uuid.UUID, jobName string, jobErr error) {
				if jobErr == nil {
					return
				}
				monitorLogger.Error("member monitor failed", "job_id", jobID, "job_name", jobName, "error", jobErr)
			}),
			gocron.AfterJobRuns(func(jobID uuid.UUID, jobName string) {
				monitorLogger.Info("completed member monitor", "job_id", jobID, "job_name", jobName)
			}),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create member monitor job: %w", err)
	}

	// Start monitoring job status in background
	go app.monitorJobStatus(j)
	return nil
}

// monitorJobStatus monitors and logs job execution status
func (app *Application) monitorJobStatus(j gocron.Job) {
	for {
		lastRun, err := j.LastRunStartedAt()
		if err != nil {
			logger.Error("failed to get last member monitor run", "error", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		if !lastRun.IsZero() {
			logger.Info("last member monitor run started", "time", lastRun.Format(time.RFC1123))
		}

		nextRun, err := j.NextRun()
		if err != nil {
			logger.Error("failed to get next member monitor run", "error", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		logger.Info("next member monitor run scheduled", "time", nextRun.Format(time.RFC1123))
		// Sleep until the next run has started
		time.Sleep(cmp.Or(max(time.Until(nextRun), 0), 1*time.Minute))
	}
}

// scheduleStatusUpdates sets up regular status message updates
func (app *Application) scheduleStatusUpdates() error {
	_, err := app.scheduler.NewJob(
		gocron.DurationJob(10*time.Second),
		gocron.NewTask(func() {
			if err := app.bot.UpdateCustomStatus(bot.NextStatusMessage()); err != nil {
				logger.Error("failed to update status message", "error", err)
			}
		}),
	)
	return err
}

// scheduleSystemdWatchdog sets up periodic systemd watchdog notifications
func (app *Application) scheduleSystemdWatchdog() error {
	if os.Getenv("NOTIFY_SOCKET") == "" {
		return nil
	}

	_, err := app.scheduler.NewJob(
		gocron.DurationJob(15*time.Second),
		gocron.NewTask(func() {
			if err := systemd.Watchdog(); err != nil {
				logger.Warn("failed to send watchdog ping", "error", err)
			}
		}),
	)

	return err
}

// shutdown gracefully shuts down all application components
func (app *Application) shutdown() {
	logger.Info("beginning graceful shutdown")

	logger.Info("notifying systemd of shutdown")
	if err := systemd.Stopping(); err != nil {
		logger.Error("failed to notify systemd of stopping", "error", err)
	}

	// Stop monitoring services
	logger.Info("stopping monitoring services")
	close(app.stopCh)

	// Shutdown scheduler
	if app.scheduler != nil {
		logger.Info("shutting down job scheduler")
		if err := app.scheduler.Shutdown(); err != nil {
			logger.Error("failed to shutdown scheduler", "error", err)
		} else {
			logger.Info("job scheduler shutdown completed")
		}
	}

	// Close bot connection
	if app.bot != nil {
		logger.Info("closing Discord bot connection")
		if err := app.bot.Close(); err != nil {
			logger.Error("failed to close the bot", "error", err)
		} else {
			logger.Info("Discord bot connection closed successfully")
		}
	}

	logger.Info("waiting 5 seconds for final cleanup")
	time.Sleep(5 * time.Second)
	logger.Info("graceful shutdown completed")
}
