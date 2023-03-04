package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/apex/log/handlers/json"
	"github.com/sol-armada/admin/bot"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/server"
	"github.com/sol-armada/admin/stores"
)

func main() {
	defer func() {
		log.Info("gracefully shutdown")
	}()
	config.SetConfigName("config")
	config.AddConfigPath(".")
	config.AddConfigPath("../")
	if err := config.ReadInConfig(); err != nil {
		log.Fatal("could not parse configuration")
		os.Exit(1)
	}

	log.SetHandler(json.New(os.Stdout))
	if config.GetBool("LOG.DEBUG") {
		log.SetLevel(log.DebugLevel)
		log.SetHandler(cli.New(os.Stdout))
		log.Debug("debug mode on")
	}

	// setup storage
	if _, err := stores.New(context.Background()); err != nil {
		log.WithError(err).Error("failed to setup storage")
	}

	// start up the bot
	b, err := bot.New()
	if err != nil {
		log.WithError(err).Error("failed to create the bot")
		return
	}

	if err := b.Open(); err != nil {
		log.WithError(err).Error("failed to start the bot")
		return
	}
	defer b.Close()

	doneMonitoring := make(chan bool, 1)
	stopMonitoring := make(chan bool, 1)
	if config.GetBoolWithDefault("FEATURES.MONITOR", false) {
		go b.Monitor(stopMonitoring, doneMonitoring)
	} else {
		doneMonitoring <- true
	}

	srv := server.New()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		stopMonitoring <- true
		if err := srv.Stop(); err != nil {
			log.WithError(err).Error("failed to stop the web server")
			return
		}
	}()

	// start the web server now that everything is running
	if err := srv.Run(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			log.WithError(err).Error("failed to start the web server")
			return
		}

		log.Info("shut down web server")
	}

	// wait for the montoring to finish
	<-doneMonitoring
}
