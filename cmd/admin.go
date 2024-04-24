package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	jsn "github.com/apex/log/handlers/json"
	"github.com/sol-armada/admin/attendance"
	"github.com/sol-armada/admin/bot"
	"github.com/sol-armada/admin/health"
	"github.com/sol-armada/admin/members"
	"github.com/sol-armada/admin/settings"
	"github.com/sol-armada/admin/stores"
)

type customFormatter struct {
	hander log.Handler
}

func (f *customFormatter) HandleLog(e *log.Entry) error {
	e.Message = fmt.Sprintf("[%s][%s] %s", time.Now().Format("15:04:05"), e.Level.String(), e.Message)
	return f.hander.HandleLog(e)
}

func init() {
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

	// monitor health of the server
	go health.Monitor()
}

func main() {
	defer func() {
		log.Info("gracefully shutdown")
	}()

	settings.SetConfigName("settings")
	settings.AddConfigPath(".")
	settings.AddConfigPath("../")
	if err := settings.ReadInConfig(); err != nil {
		log.Fatal("could not parse configuration")
		os.Exit(1)
	}

	log.SetHandler(jsn.New(os.Stdout))
	if settings.GetBool("LOG.DEBUG") {
		log.SetLevel(log.DebugLevel)
		if settings.GetBool("LOG.CLI") {
			handler := &customFormatter{hander: cli.New(os.Stdout)}
			log.SetHandler(handler)
		}
		log.Debug("debug mode on")
	}

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
	defer b.Close()

	doneMonitoring := make(chan bool, 1)
	stopMonitoring := make(chan bool, 1)
	if settings.GetBool("FEATURES.MONITOR.ENABLE") {
		go b.UserMonitor(stopMonitoring, doneMonitoring)
	} else {
		doneMonitoring <- true
	}
	defer func() {
		doneMonitoring <- true
		stopMonitoring <- true
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	stopMonitoring <- true
}
