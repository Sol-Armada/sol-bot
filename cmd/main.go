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

var environment string = ""

var logger *slog.Logger

func init() {
	if environment == "staging" {
		settings.SetConfigName("settings.staging")
	} else {
		settings.SetConfigName("settings")
	}

	// setup settings
	settings.AddConfigPath(".")
	settings.AddConfigPath("../")
	settings.AddConfigPath("/etc/solbot/")
	if err := settings.ReadInConfig(); err != nil {
		fmt.Printf("could not parse configuration: %v\n", err)
		os.Exit(1)
	}

	// setup the logger
	opts := &slog.HandlerOptions{
		AddSource: true,
	}

	if settings.GetBool("LOG.DEBUG") {
		opts.Level = slog.LevelDebug
	}

	if settings.GetBool("LOG.CLI") {
		logger = slog.New(slog.NewTextHandler(os.Stdout, opts))
	} else {
		f, err := os.OpenFile(settings.GetStringWithDefault("LOG.FILE", "/var/log/solbot/solbot.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("Failed to open log file: %v\n", err)
			os.Exit(1)
		}
		logger = slog.New(slog.NewJSONHandler(f, opts))
	}

	// Set as default logger
	slog.SetDefault(logger)

	if settings.GetBool("LOG.DEBUG") {
		logger.Debug("debug mode on")
	}

	// setup storage
	host := settings.GetStringWithDefault("mongo.host", "localhost")
	port := settings.GetIntWithDefault("mongo.port", 27017)
	username := settings.GetString("MONGO.USERNAME")
	pswd := strings.ReplaceAll(settings.GetString("MONGO.PASSWORD"), "@", `%40`)
	database := settings.GetStringWithDefault("MONGO.DATABASE", "org")
	replicaSetName := settings.GetString("MONGO.REPLICA_SET_NAME")
	ctx := context.Background()

	if _, err := stores.New(ctx, host, port, username, pswd, database, replicaSetName); err != nil {
		logger.Error("failed to create storage client", "error", err)
		os.Exit(1)
	}

	setups := map[string]func() error{
		"members":    members.Setup,
		"attendance": attendance.Setup,
		"activity":   activity.Setup,
		"tokens":     tokens.Setup,
		"config":     config.Setup,
		"raffles":    raffles.Setup,
	}

	for name, setup := range setups {
		logger.Debug("setting up store", "name", name)

		if err := setup(); err != nil {
			logger.Error("failed to setup store", "name", name, "error", err)
			os.Exit(1)
		}
	}

	// monitor health of the server
	go health.Monitor()
}

func main() {
	defer func() {
		// Notify systemd we're stopping
		if err := systemd.Stopping(); err != nil {
			logger.Error("failed to notify systemd of stopping", "error", err)
		}
		logger.Info("gracefully shutdown")
	}()

	// Notify systemd of our status during startup
	if err := systemd.Status("Starting up..."); err != nil {
		logger.Warn("failed to set systemd status", "error", err)
	}

	// start up the bot
	b, err := bot.New()
	if err != nil {
		logger.Error("failed to create the bot", "error", err)
		return
	}

	if err := systemd.Status("Setting up bot..."); err != nil {
		logger.Warn("failed to set systemd status", "error", err)
	}

	if err := b.Setup(); err != nil {
		logger.Error("failed to start the bot", "error", err)
		return
	}

	// Notify systemd that we're ready
	if err := systemd.Ready(); err != nil {
		logger.Warn("failed to notify systemd ready", "error", err)
	}

	if err := systemd.Status("Bot ready and serving"); err != nil {
		logger.Warn("failed to set systemd status", "error", err)
	}

	s, err := gocron.NewScheduler()
	if err != nil {
		logger.Error("failed to create scheduler", "error", err)
		return
	}

	// Start the scheduler
	s.Start()

	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	}

	if settings.GetBool("LOG.DEBUG") {
		opts.Level = slog.LevelDebug
		slog.Debug("debug mode on")
	}

	monitorLogger := slog.New(slog.NewTextHandler(os.Stdout, opts))

	if !settings.GetBool("LOG.CLI") {
		f, err := os.OpenFile(settings.GetStringWithDefault("LOG.MEMBER_MONITOR_FILE", "/var/log/solbot/mm.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			logger.Error("failed to open member monitor log file", "error", err)
			os.Exit(1)
		}
		monitorLogger = slog.New(slog.NewJSONHandler(f, opts))
	}

	if settings.GetBool("FEATURES.MONITOR.ENABLE") {
		slog.Info("scheduling member monitor")
		j, err := s.NewJob(
			gocron.CronJob("*/30 * * * *", false),
			gocron.NewTask(func(ctx context.Context) error {
				return bot.MemberMonitor(ctx, monitorLogger)
			}),
			gocron.WithSingletonMode(gocron.LimitModeReschedule),
			gocron.WithEventListeners(
				gocron.BeforeJobRuns(
					func(jobID uuid.UUID, jobName string) {
						monitorLogger.Info("starting member monitor", "job_id", jobID, "job_name", jobName)
					},
				),
				gocron.AfterJobRunsWithError(func(jobID uuid.UUID, jobName string, jobErr error) {
					if jobErr != nil {
						monitorLogger.Error("member monitor failed", "job_id", jobID, "job_name", jobName, "error", jobErr)
					}
				}),
				gocron.AfterJobRuns(func(jobID uuid.UUID, jobName string) {
					monitorLogger.Info("completed member monitor", "job_id", jobID, "job_name", jobName)
				}),
			),
		)
		if err != nil {
			logger.Error("failed to schedule member monitor", "error", err)
			return
		}

		go func() {
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
				// sleep until the next run has started
				time.Sleep(time.Until(nextRun))
			}
		}()
	}

	stopAttendanceMonitor := make(chan bool, 1)
	if settings.GetBool("FEATURES.ATTENDANCE.MONITOR") { // only enable if attendance is enabled
		go bot.MonitorAttendance(context.Background(), logger, stopAttendanceMonitor)
	}

	if _, err := s.NewJob(
		gocron.DurationJob(10*time.Second),
		gocron.NewTask(func() {
			if err := b.UpdateCustomStatus(bot.NextStatusMessage()); err != nil {
				logger.Error("failed to update status message", "error", err)
			}
		}),
	); err != nil {
		logger.Error("failed to schedule status message updater", "error", err)
		return
	}

	// Optional: Start systemd watchdog if enabled
	watchdogStop := make(chan bool, 1)
	go func() {
		ticker := time.NewTicker(15 * time.Second) // Send watchdog every 15 seconds
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := systemd.Watchdog(); err != nil {
					logger.Debug("failed to send watchdog ping", "error", err)
				}
			case <-watchdogStop:
				return
			}
		}
	}()

	defer func() {
		logger.Info("shutting down")
		if err := s.Shutdown(); err != nil {
			logger.Error("failed to shutdown scheduler", "error", err)
		}
		if err := b.Close(); err != nil {
			logger.Error("failed to close the bot", "error", err)
		}
		stopAttendanceMonitor <- true
		watchdogStop <- true
		time.Sleep(20 * time.Second)
		logger.Info("shutdown complete")
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	// capture signal from supervisord
	signal.Notify(c, syscall.SIGTERM)
	<-c
}
