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
	"github.com/sol-armada/admin/bot"
	"github.com/sol-armada/admin/cache"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/health"
	"github.com/sol-armada/admin/stores"
)

type customFormatter struct {
	hander log.Handler
}

func (f *customFormatter) HandleLog(e *log.Entry) error {
	e.Message = fmt.Sprintf("[%s][%s] %s", time.Now().Format("15:04:05"), e.Level.String(), e.Message)
	return f.hander.HandleLog(e)
}

func main() {
	defer func() {
		log.Info("gracefully shutdown")
	}()

	config.SetConfigName("settings")
	config.AddConfigPath(".")
	config.AddConfigPath("../")
	if err := config.ReadInConfig(); err != nil {
		log.Fatal("could not parse configuration")
		os.Exit(1)
	}

	log.SetHandler(jsn.New(os.Stdout))
	if config.GetBool("LOG.DEBUG") {
		log.SetLevel(log.DebugLevel)
		if config.GetBool("LOG.CLI") {
			handler := &customFormatter{hander: cli.New(os.Stdout)}
			log.SetHandler(handler)
		}
		log.Debug("debug mode on")
	}

	// setup storage
	host := config.GetStringWithDefault("mongo.host", "localhost")
	port := config.GetIntWithDefault("mongo.port", 27017)
	username := config.GetString("MONGO.USERNAME")
	pswd := strings.ReplaceAll(config.GetString("MONGO.PASSWORD"), "@", `%40`)
	database := config.GetStringWithDefault("MONGO.DATABASE", "org")
	if err := stores.Setup(context.Background(), host, port, username, pswd, database); err != nil {
		log.WithError(err).Error("failed to setup storage")
	}
	cache.Setup()

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
	if config.GetBoolWithDefault("FEATURES.MONITOR", false) {
		go b.UserMonitor(stopMonitoring, doneMonitoring)
	} else {
		doneMonitoring <- true
	}
	defer func() {
		doneMonitoring <- true
		stopMonitoring <- true
	}()

	// monitor health of the server
	go health.Monitor()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		stopMonitoring <- true
	}()
}
