package main

import (
	"os"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/bwmarrin/discordgo"
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

	// register commands
	if _, err := b.ApplicationCommandCreate(config.GetString("DISCORD.CLIENT_ID"), config.GetString("DISCORD.GUILD_ID"), &discordgo.ApplicationCommand{
		Name:        "attendance",
		Description: "Get your Event Attendence count",
	}); err != nil {
		log.WithError(err).Error("failed creating attendance command")
		return
	}
	if _, err := b.ApplicationCommandCreate(config.GetString("DISCORD.CLIENT_ID"), config.GetString("DISCORD.GUILD_ID"), &discordgo.ApplicationCommand{
		Name:        "event",
		Description: "Event Actions",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "attendance",
				Description: "Take attendance of an Event going on now",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
		},
	}); err != nil {
		log.WithError(err).Error("failed creating attendance command")
		return
	}

	channels, err := b.GuildChannels(config.GetString("DISCORD.GUILD_ID"))
	if err != nil {
		log.WithError(err).Error("getting active threads")
		return
	}

	for _, channel := range channels {
		if err := b.State.ChannelAdd(channel); err != nil {
			log.WithError(err).Error("adding channel to state")
			return
		}
	}

	defer b.Close()

	// go b.Monitor()

	if err := server.Run(); err != nil {
		log.WithError(err).Error("failed to start the web server")
		return
	}
}
