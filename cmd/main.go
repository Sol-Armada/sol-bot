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
	cfg := loadConfig()
	logger = setupLogger(cfg)

	if err := initializeServices(cfg); err != nil {
		logger.Error("failed to initialize services", "error", err)
		os.Exit(1)
	}

	go health.Monitor()
}

// loadConfig initializes and loads application configuration
func loadConfig() *Config {
	configName := "settings"
	if environment == "staging" {
		configName = "settings.staging"
	}
	settings.SetConfigName(configName)

	// Setup settings paths
	settings.AddConfigPath(".")
	settings.AddConfigPath("../")
	settings.AddConfigPath("/etc/solbot/")

	if err := settings.ReadInConfig(); err != nil {
		fmt.Printf("could not parse configuration: %v\n", err)
		os.Exit(1)
	}

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
	opts := &slog.HandlerOptions{
		AddSource: true,
	}

	if cfg.Debug {
		opts.Level = slog.LevelDebug
	}

	// Create CLI logger if configured for CLI output
	if cfg.CLI {
		log := slog.New(slog.NewTextHandler(os.Stdout, opts))
		slog.SetDefault(log)
		return log
	}

	// Create file logger for non-CLI output
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

	return log
}

// initializeServices sets up database connection and initializes all services
func initializeServices(cfg *Config) error {
	ctx := context.Background()

	// Initialize database connection
	if _, err := stores.New(
		ctx,
		cfg.MongoConfig.Host,
		cfg.MongoConfig.Port,
		cfg.MongoConfig.Username,
		cfg.MongoConfig.Password,
		cfg.MongoConfig.Database,
		cfg.MongoConfig.ReplicaSetName,
	); err != nil {
		return fmt.Errorf("failed to create storage client: %w", err)
	}

	// Initialize all services
	services := map[string]func() error{
		"members":    members.Setup,
		"attendance": attendance.Setup,
		"activity":   activity.Setup,
		"tokens":     tokens.Setup,
		"config":     config.Setup,
		"raffles":    raffles.Setup,
	}

	for name, setup := range services {
		if err := setup(); err != nil {
			return fmt.Errorf("failed to setup %s service: %w", name, err)
		}
	}

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

	cfg := loadConfig()
	app := &Application{
		cfg:    cfg,
		logger: logger,
		stopCh: make(chan struct{}),
	}

	defer app.shutdown()

	if err := app.start(); err != nil {
		logger.Error("failed to start application", "error", err)
		os.Exit(1)
	}

	// Wait for shutdown signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	<-c
}

// start initializes and starts all application components
func (app *Application) start() error {
	if app.cfg.Features.SystemdIntegration {
		if err := systemd.Status("Starting up..."); err != nil {
			logger.Warn("failed to set systemd status", "error", err)
		}
	}

	// Initialize bot
	if err := app.initializeBot(); err != nil {
		return fmt.Errorf("failed to initialize bot: %w", err)
	}

	// Initialize scheduler
	if err := app.initializeScheduler(); err != nil {
		return fmt.Errorf("failed to initialize scheduler: %w", err)
	}

	// Start monitoring services
	app.startMonitoringServices()

	if app.cfg.Features.SystemdIntegration {
		if err := systemd.Ready(); err != nil {
			logger.Warn("failed to notify systemd ready", "error", err)
		}
		if err := systemd.Status("Bot ready and serving"); err != nil {
			logger.Warn("failed to set systemd status", "error", err)
		}
	}

	return nil
}

// initializeBot creates and sets up the Discord bot
func (app *Application) initializeBot() error {
	var err error
	app.bot, err = bot.New()
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}

	if app.cfg.Features.SystemdIntegration {
		if err := systemd.Status("Setting up bot..."); err != nil {
			logger.Warn("failed to set systemd status", "error", err)
		}
	}

	if err := app.bot.Setup(); err != nil {
		return fmt.Errorf("failed to setup bot: %w", err)
	}

	return nil
}

// initializeScheduler creates and configures the job scheduler
func (app *Application) initializeScheduler() error {
	var err error
	app.scheduler, err = gocron.NewScheduler()
	if err != nil {
		return fmt.Errorf("failed to create scheduler: %w", err)
	}

	app.scheduler.Start()

	// Schedule member monitoring if enabled
	if app.cfg.Features.MonitorEnable {
		if err := app.scheduleMemberMonitor(); err != nil {
			return fmt.Errorf("failed to schedule member monitor: %w", err)
		}
	}

	// Schedule status message updates
	if err := app.scheduleStatusUpdates(); err != nil {
		return fmt.Errorf("failed to schedule status updates: %w", err)
	}

	if app.cfg.Features.SystemdIntegration {
		if err := app.scheduleSystemdWatchdog(); err != nil {
			return fmt.Errorf("failed to schedule systemd watchdog: %w", err)
		}
	}

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
		go bot.MonitorAttendance(context.Background(), logger, app.stopCh)
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
	if app.cfg.Features.SystemdIntegration {
		if err := systemd.Stopping(); err != nil {
			logger.Error("failed to notify systemd of stopping", "error", err)
		}
	}

	// Stop monitoring services
	close(app.stopCh)

	// Shutdown scheduler
	if app.scheduler != nil {
		if err := app.scheduler.Shutdown(); err != nil {
			logger.Error("failed to shutdown scheduler", "error", err)
		}
	}

	// Close bot connection
	if app.bot != nil {
		if err := app.bot.Close(); err != nil {
			logger.Error("failed to close the bot", "error", err)
		}
	}

	time.Sleep(5 * time.Second)
}
