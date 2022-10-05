package main

import (
	"os"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/sol-armada/admin/bot"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/server"
	"github.com/sol-armada/admin/users"
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

	users.GetStorage()
	if err := users.LoadAdmins(); err != nil {
		log.WithError(err).Error("failed to load admins")
		return
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

	go b.Monitor()

	if err := server.Run(); err != nil {
		log.WithError(err).Error("failed to start the web server")
		return
	}
}
