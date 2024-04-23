package bot

import (
	"context"
	"fmt"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/members"
	"github.com/sol-armada/admin/ranks"
	"github.com/sol-armada/admin/utils"
)

func giveMeritCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("merit command")

	user, err := members.Get(i.Member.User.ID)
	if err != nil {
		return errors.Wrap(err, "getting user from storage for merit command")
	}

	if user.Rank > ranks.GetRankByName(config.GetStringWithDefault("FEATURES.ATTENDANCE.MIN_RANK", "admiral")) {
		logger.Debug("invalid permissions")

		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "You do not have permission to use this command",
			},
		}); err != nil {
			return errors.Wrap(err, "responding to merit command: invalid permissions")
		}

		return nil
	}

	data := i.ApplicationCommandData()

	receivingDiscordUser := data.Options[0].UserValue(s)

	receivingMember, err := members.Get(receivingDiscordUser.ID)
	if err != nil {
		return errors.Wrap(err, "getting receiving member")
	}

	if err := receivingMember.GiveMerit(data.Options[1].StringValue(), user); err != nil {
		return errors.Wrap(err, "giving member merit")
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Gave %s the merit!", receivingDiscordUser.Username),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		return errors.Wrap(err, "responding to merit command")
	}

	return nil
}
