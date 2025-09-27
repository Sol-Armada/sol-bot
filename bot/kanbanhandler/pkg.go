package kanbanhandler

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

var autoCompletes = map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error{}

var buttons = map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error{}

var modals = map[string]func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error{}

func Setup() (*discordgo.ApplicationCommand, error) {
	options := []*discordgo.ApplicationCommandOption{}
	return &discordgo.ApplicationCommand{
		Name:        "kanban",
		Description: "Manage kanban boards",
		Type:        discordgo.ChatApplicationCommand,
		Options:     options,
	}, nil
}
