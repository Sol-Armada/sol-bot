package attendancehandler

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/utils"
)

func addNameCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("add attendance name command")

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	data := i.Interaction.ApplicationCommandData()

	name := data.Options[0].Options[0].StringValue()

	if err := settings.NewAttendanceName(name); err != nil {
		return err
	}

	_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "Name \"" + name + "\" added successfully",
	})

	return err
}
