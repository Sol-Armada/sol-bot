package bot

import (
	"context"
	"fmt"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/members"
	"github.com/sol-armada/admin/settings"
	"github.com/sol-armada/admin/utils"
)

func giveDemeritCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("demerit command")

	if !utils.StringSliceContainsOneOf(i.Member.Roles, settings.GetStringSlice("FEATURES.MERIT.ALLOWED_ROLES")) {
		logger.Debug("invalid permissions")

		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "You do not have permission to use this command",
			},
		}); err != nil {
			return errors.Wrap(err, "responding to demerit command: invalid permissions")
		}

		return nil
	}

	data := i.ApplicationCommandData()

	receivingDiscordUser := data.Options[0].UserValue(s)

	receivingMember, err := members.Get(receivingDiscordUser.ID)
	if err != nil {
		return errors.Wrap(err, "getting receiving member")
	}

	givingMember, err := members.Get(i.Member.User.ID)
	if err != nil {
		return errors.Wrap(err, "getting member from storage for demerit command")
	}

	if err := receivingMember.GiveDemerit(data.Options[1].StringValue(), givingMember); err != nil {
		return errors.Wrap(err, "giving member demerit")
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Gave %s the demerit!", receivingDiscordUser.Username),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		return errors.Wrap(err, "responding to demerit command")
	}

	return nil
}
