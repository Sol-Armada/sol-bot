package main

import (
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
	"github.com/sol-armada/sol-bot/health"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/raffles"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/stores"
	"github.com/sol-armada/sol-bot/systemd"
	"github.com/sol-armada/sol-bot/tokens"
)

var (
	environment string = ""
	logger      *slog.Logger
)

// Config holds all application configuration
type Config struct {
	Environment          string
	Debug                bool
	CLI                  bool
	LogFile              string
	MemberMonitorLogFile string
	MongoConfig          MongoConfig
	Features             FeatureConfig
}

type MongoConfig struct {
	Host           string
	Port           int
	Username       string
	Password       string
	Database       string
	ReplicaSetName string
}

type FeatureConfig struct {
	MonitorEnable      bool
	AttendanceMonitor  bool
	SystemdIntegration bool
}

func init() {
	fmt.Println("Starting sol-bot initialization...")

	cfg := loadConfig()
	logger = setupLogger(cfg)

	logger.Info("sol-bot starting up",
		"environment", cfg.Environment,
		"debug", cfg.Debug,
		"cli", cfg.CLI,
		"version", "unknown") // You could add a version variable later

	logger.Info("configuration loaded successfully",
		"mongo_host", cfg.MongoConfig.Host,
		"mongo_port", cfg.MongoConfig.Port,
		"mongo_database", cfg.MongoConfig.Database,
		"features_monitor_enable", cfg.Features.MonitorEnable,
		"features_attendance_monitor", cfg.Features.AttendanceMonitor,
		"features_systemd_integration", cfg.Features.SystemdIntegration)

	if err := initializeServices(cfg); err != nil {
		logger.Error("failed to initialize services", "error", err)
		os.Exit(1)
	}

	logger.Info("health monitor starting")
	go health.Monitor()
}

// loadConfig initializes and loads application configuration
func loadConfig() *Config {
	fmt.Printf("Loading configuration for environment: %s\n", environment)

	configName := "settings"
	if environment == "staging" {
		configName = "settings.staging"
		fmt.Printf("Using staging configuration: %s\n", configName)
	}
	settings.SetConfigName(configName)

	// Setup settings paths
	configPaths := []string{".", "../", "/etc/solbot/"}
	for _, path := range configPaths {
		settings.AddConfigPath(path)
		fmt.Printf("Adding config search path: %s\n", path)
	}

	fmt.Printf("Attempting to read configuration file: %s\n", configName)
	if err := settings.ReadInConfig(); err != nil {
		fmt.Printf("could not parse configuration: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Configuration loaded successfully\n")

	return &Config{
		Environment:          environment,
		Debug:                settings.GetBool("LOG.DEBUG"),
		CLI:                  settings.GetBool("LOG.CLI"),
		LogFile:              settings.GetStringWithDefault("LOG.FILE", "/var/log/solbot/solbot.log"),
		MemberMonitorLogFile: settings.GetStringWithDefault("LOG.MEMBER_MONITOR_FILE", "/var/log/solbot/mm.log"),
		MongoConfig: MongoConfig{
			Host:           settings.GetStringWithDefault("mongo.host", "localhost"),
			Port:           settings.GetIntWithDefault("mongo.port", 27017),
			Username:       settings.GetString("MONGO.USERNAME"),
			Password:       strings.ReplaceAll(settings.GetString("MONGO.PASSWORD"), "@", `%40`),
			Database:       settings.GetStringWithDefault("MONGO.DATABASE", "org"),
			ReplicaSetName: settings.GetString("MONGO.REPLICA_SET_NAME"),
		},
		Features: FeatureConfig{
			MonitorEnable:      settings.GetBool("FEATURES.MONITOR.ENABLE"),
			AttendanceMonitor:  settings.GetBool("FEATURES.ATTENDANCE.MONITOR"),
			SystemdIntegration: settings.GetBoolWithDefault("FEATURES.SYSTEMD.ENABLE", true),
		},
	}
}

// setupLogger creates and configures the application logger
func setupLogger(cfg *Config) *slog.Logger {
	fmt.Printf("Setting up logger - Debug: %v, CLI: %v, LogFile: %s\n", cfg.Debug, cfg.CLI, cfg.LogFile)

	opts := &slog.HandlerOptions{
		AddSource: true,
	}

	if cfg.Debug {
		opts.Level = slog.LevelDebug
		fmt.Printf("Debug logging enabled\n")
	}

	// Create CLI logger if configured for CLI output
	if cfg.CLI {
		fmt.Printf("Using CLI logger (stdout)\n")
		log := slog.New(slog.NewTextHandler(os.Stdout, opts))
		slog.SetDefault(log)
		return log
	}

	// Create file logger for non-CLI output
	fmt.Printf("Using file logger: %s\n", cfg.LogFile)
	f, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open log file: %v\n", err)
		os.Exit(1)
	}

	log := slog.New(slog.NewJSONHandler(f, opts))
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
	logger.Info("initializing database connection",
		"host", cfg.MongoConfig.Host,
		"port", cfg.MongoConfig.Port,
		"database", cfg.MongoConfig.Database,
		"replica_set", cfg.MongoConfig.ReplicaSetName)

	if _, err := stores.New(
		ctx,
		cfg.MongoConfig.Host,
		cfg.MongoConfig.Port,
		cfg.MongoConfig.Username,
		cfg.MongoConfig.Password,
		cfg.MongoConfig.Database,
		cfg.MongoConfig.ReplicaSetName,
	); err != nil {
		logger.Error("failed to create storage client", "error", err)
		return fmt.Errorf("failed to create storage client: %w", err)
	}
	logger.Info("database connection established successfully")

	// Initialize all services
	services := map[string]func() error{
		"members":    members.Setup,
		"attendance": attendance.Setup,
		"activity":   activity.Setup,
		"tokens":     tokens.Setup,
		"config":     config.Setup,
		"raffles":    raffles.Setup,
	}

	logger.Info("initializing services", "count", len(services))
	for name, setup := range services {
		logger.Info("setting up service", "service", name)
		if err := setup(); err != nil {
			logger.Error("failed to setup service", "service", name, "error", err)
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

	if app.cfg.Features.SystemdIntegration {
		logger.Info("systemd integration enabled, setting startup status")
		if err := systemd.Status("Starting up..."); err != nil {
			logger.Warn("failed to set systemd status", "error", err)
		}
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

	// Start monitoring services
	logger.Info("starting monitoring services")
	app.startMonitoringServices()
	logger.Info("monitoring services started")

	if app.cfg.Features.SystemdIntegration {
		logger.Info("notifying systemd that application is ready")
		if err := systemd.Ready(); err != nil {
			logger.Warn("failed to notify systemd ready", "error", err)
		}
		if err := systemd.Status("Bot ready and serving"); err != nil {
			logger.Warn("failed to set systemd status", "error", err)
		}
		logger.Info("systemd notifications sent successfully")
	}

	logger.Info("application startup completed successfully")
	return nil
}

// initializeBot creates and sets up the Discord bot
func (app *Application) initializeBot() error {
	logger.Info("creating new bot instance")
	var err error
	app.bot, err = bot.New()
	if err != nil {
		logger.Error("failed to create bot instance", "error", err)
		return fmt.Errorf("failed to create bot: %w", err)
	}
	logger.Info("bot instance created successfully")

	if app.cfg.Features.SystemdIntegration {
		if err := systemd.Status("Setting up bot..."); err != nil {
			logger.Warn("failed to set systemd status", "error", err)
		}
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

		if app.cfg.Features.SystemdIntegration {
			status := fmt.Sprintf("Initializing bot (attempt %d/%d)...", attempt, maxAttempts)
			if err := systemd.Status(status); err != nil {
				logger.Warn("failed to set systemd status", "error", err)
			}
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

		if app.cfg.Features.SystemdIntegration {
			status := fmt.Sprintf("Initialization failed, retrying in %v...", delay)
			if err := systemd.Status(status); err != nil {
				logger.Warn("failed to set systemd status", "error", err)
			}
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
	if app.cfg.Features.MonitorEnable {
		logger.Info("scheduling member monitor job")
		if err := app.scheduleMemberMonitor(); err != nil {
			logger.Error("failed to schedule member monitor", "error", err)
			return fmt.Errorf("failed to schedule member monitor: %w", err)
		}
		logger.Info("member monitor job scheduled successfully")
	} else {
		logger.Info("member monitoring disabled")
	}

	// Schedule status message updates
	logger.Info("scheduling status update job")
	if err := app.scheduleStatusUpdates(); err != nil {
		logger.Error("failed to schedule status updates", "error", err)
		return fmt.Errorf("failed to schedule status updates: %w", err)
	}
	logger.Info("status update job scheduled successfully")

	if app.cfg.Features.SystemdIntegration {
		logger.Info("scheduling systemd watchdog job")
		if err := app.scheduleSystemdWatchdog(); err != nil {
			logger.Error("failed to schedule systemd watchdog", "error", err)
			return fmt.Errorf("failed to schedule systemd watchdog: %w", err)
		}
		logger.Info("systemd watchdog job scheduled successfully")
	}

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

	if app.cfg.CLI {
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}

	f, err := os.OpenFile(app.cfg.MemberMonitorLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Error("failed to open member monitor log file", "error", err)
		os.Exit(1)
	}

	return slog.New(slog.NewJSONHandler(f, opts))
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
		lastRun, err := j.LastRun()
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
		time.Sleep(time.Until(nextRun))
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

// startMonitoringServices starts attendance monitoring and systemd watchdog
func (app *Application) startMonitoringServices() {
	// Start attendance monitoring if enabled
	if app.cfg.Features.AttendanceMonitor {
		logger.Info("starting attendance monitoring service")
		go bot.MonitorAttendance(context.Background(), logger, app.stopCh)
		logger.Info("attendance monitoring service started")
	} else {
		logger.Info("attendance monitoring disabled")
	}
}

// scheduleSystemdWatchdog sets up periodic systemd watchdog notifications
func (app *Application) scheduleSystemdWatchdog() error {
	if os.Getenv("NOTIFY_SOCKET") == "" {
		return nil
	}

	_, err := app.scheduler.NewJob(
		gocron.DurationJob(15*time.Second),
		gocron.NewTask(func() {
			logger.Debug("sending systemd watchdog ping")
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

	if app.cfg.Features.SystemdIntegration {
		logger.Info("notifying systemd of shutdown")
		if err := systemd.Stopping(); err != nil {
			logger.Error("failed to notify systemd of stopping", "error", err)
		}
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
