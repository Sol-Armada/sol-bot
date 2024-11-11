package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	jsn "github.com/apex/log/handlers/json"
	"github.com/sol-armada/sol-bot/activity"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/bot"
	"github.com/sol-armada/sol-bot/config"
	"github.com/sol-armada/sol-bot/health"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/stores"
)

type customFormatter struct {
	hander log.Handler
}

func (f *customFormatter) HandleLog(e *log.Entry) error {
	e.Message = fmt.Sprintf("[%s][%s] %s", time.Now().Format("15:04:05"), e.Level.String(), e.Message)
	return f.hander.HandleLog(e)
}

func init() {
	// setup settings
	settings.SetConfigName("settings")
	settings.AddConfigPath(".")
	settings.AddConfigPath("../")
	settings.AddConfigPath("/etc/solbot/")
	if err := settings.ReadInConfig(); err != nil {
		log.Fatal("could not parse configuration")
		os.Exit(1)
	}

	// setup the logger
	if settings.GetBool("LOG.CLI") {
		handler := &customFormatter{hander: cli.New(os.Stdout)}
		log.SetHandler(handler)
	} else {
		f, err := os.OpenFile(settings.GetStringWithDefault("LOG.FILE", "/var/log/solbot/solbot.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err.Error())
			os.Exit(1)
		}
		log.SetHandler(jsn.New(f))
	}

	if settings.GetBool("LOG.DEBUG") {
		log.SetLevel(log.DebugLevel)
		log.Debug("debug mode on")
	}

	// setup storage
	host := settings.GetStringWithDefault("mongo.host", "localhost")
	port := settings.GetIntWithDefault("mongo.port", 27017)
	username := settings.GetString("MONGO.USERNAME")
	pswd := strings.ReplaceAll(settings.GetString("MONGO.PASSWORD"), "@", `%40`)
	database := settings.GetStringWithDefault("MONGO.DATABASE", "org")
	ctx := context.Background()

	if _, err := stores.New(ctx, host, port, username, pswd, database); err != nil {
		log.WithError(err).Error("failed to create storage client")
		os.Exit(1)
	}

	if err := members.Setup(); err != nil {
		log.WithError(err).Error("failed to setup members")
		os.Exit(1)
	}

	if err := attendance.Setup(); err != nil {
		log.WithError(err).Error("failed to setup attendance")
		os.Exit(1)
	}

	if err := activity.Setup(); err != nil {
		log.WithError(err).Error("failed to setup activity")
		os.Exit(1)
	}

	if err := config.Setup(); err != nil {
		log.WithError(err).Error("failed to setup config")
		os.Exit(1)
	}

	// monitor health of the server
	go health.Monitor()
}

func main() {
	defer func() {
		log.Info("gracefully shutdown")
	}()

	// start up the bot
	b, err := bot.New()
	if err != nil {
		log.WithError(err).Error("failed to create the bot")
		return
	}

	if err := b.Setup(); err != nil {
		log.WithError(err).Error("failed to start the bot")
		return
	}

	stopMemberMonitor := make(chan bool, 1)
	if settings.GetBool("FEATURES.MONITOR.ENABLE") {
		go bot.MemberMonitor(stopMemberMonitor)
	}
	stopAttendanceMonitor := make(chan bool, 1)
	if settings.GetBool("FEATURES.ATTENDANCE.MONITOR") { // only enable if attendance is enabled
		go bot.MonitorAttendance(stopAttendanceMonitor)
	}
	defer func() {
		log.Info("shutting down")
		if err := b.Close(); err != nil {
			log.WithError(err).Error("failed to close the bot")
		}
		stopMemberMonitor <- true
		stopAttendanceMonitor <- true
		time.Sleep(20 * time.Second)
		log.Info("shutdown complete")
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	// capture signal from supervisord
	signal.Notify(c, syscall.SIGTERM)
	<-c
}
