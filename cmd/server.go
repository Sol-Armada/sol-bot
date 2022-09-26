package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/router"
)

func main() {
	config.SetConfigName("config")
	config.AddConfigPath(".")
	config.AddConfigPath("../")
	if err := config.ReadInConfig(); err != nil {
		log.Fatal("could not parse configuration")
		os.Exit(1)
	}

	log.SetHandler(cli.New(os.Stdout))
	if config.GetBool("LOG.DEBUG") {
		log.SetLevel(log.DebugLevel)
		log.Debug("debug mode on")
	}

	port := fmt.Sprintf(":%d", config.GetIntWithDefault("SERVER.PORT", 8080))
	srv := &http.Server{
		Addr:    port,
		Handler: router.Router(),
	}

	log.WithField("port", port).Info("starting server")
	if err := srv.ListenAndServe(); err != nil {
		log.WithError(err).Error("serve website")
	}
}
