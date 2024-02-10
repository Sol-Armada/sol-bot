package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	jsn "github.com/apex/log/handlers/json"
	"github.com/sol-armada/admin/bot"
	"github.com/sol-armada/admin/cache"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/events"
	"github.com/sol-armada/admin/health"
	"github.com/sol-armada/admin/onboarding"
	"github.com/sol-armada/admin/server"
	"github.com/sol-armada/admin/stores"
	"github.com/sol-armada/admin/users"
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
	if err := stores.Setup(context.Background()); err != nil {
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

	// events
	if config.GetBoolWithDefault("FEATURES.EVENTS.ENABLED", false) {
		log.Info("using events feature")

		if err := events.Setup(b); err != nil {
			log.WithError(err).Error("setting up onboarding")
		}
	}

	// onboarding
	if config.GetBool("FEATURES.ONBOARDING") {
		log.Info("using onboarding feature")

		if err := onboarding.Setup(b); err != nil {
			log.WithError(err).Error("setting up onboarding")
		}
	}

	// monitor health of the server
	go health.Monitor()

	// start syncing the stores
	go syncStores(doneMonitoring)

	// start the web server now that everything is running
	srv, err := server.New()
	if err != nil {
		log.WithError(err).Error("starting web server")
		return
	}
	if err := srv.Start(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			log.WithError(err).Error("failed to start the web server")
			return
		}

		log.Info("shut down web server")
	}

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
}

func syncStores(stop chan bool) {
	logger := log.WithField("func", "syncStores")
	logger.Debug("syncing stores")

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	logger.Debug("starting syncing stores")
	for {
		select {
		case <-stop:
			logger.Debug("stopping syncing stores")
			return
		case <-ticker.C:
		}

		logger.Info("syncing users")

		usersSynced := 0

		// get all users from cache
		usersRaw := cache.Cache.GetUsers()

		// convert to slice of users.User
		for _, v := range usersRaw {
			logger = logger.WithFields(log.Fields{
				"user":  v["id"],
				"count": usersSynced,
			})
			logger.Debug("syncing user")
			userRaw, err := json.Marshal(v)
			if err != nil {
				logger.WithError(err).Error("marshalling raw user from cache")
				continue
			}
			usr := users.User{}
			if err := json.Unmarshal(userRaw, &usr); err != nil {
				logger.WithError(err).Error("unmarshalling user from cache")
				continue
			}

			if err := stores.Users.Update(usr.ID, usr); err != nil {
				logger.WithError(err).Error("updating user")
			}

			usersSynced++
		}

		logger.WithField("users", usersSynced).Info("synced users")
	}
}
