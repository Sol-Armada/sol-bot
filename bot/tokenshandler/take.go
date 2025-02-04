package tokenshandler

import (
	"context"
	"errors"
	"fmt"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/tokens"
	"github.com/sol-armada/sol-bot/utils"
)

func takeCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("take command handler")

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	var member *members.Member
	var amount int = 0
	var reason tokens.Reason
	var comment string

	options := i.ApplicationCommandData().Options[0].Options
	for _, option := range options {
		switch option.Name {
		case "member":
			discordMember := option.UserValue(s)
			m, err := members.Get(discordMember.ID)
			if err != nil {
				return err
			}

			member = m
		case "amount":
			amount = int(option.IntValue())
		case "reason":
			reason = tokens.Reason(option.StringValue())
		case "comment":
			comment = option.StringValue()
		}
	}

	if amount <= 0 {
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "Amount must be greater than 0",
		})
		return err
	}

	if reason == "" {
		return errors.New("reason is required")
	}

	if err := tokens.New(member.Id, amount*-1, reason, nil, &comment).Save(); err != nil {
		return err
	}

	_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Flags:   discordgo.MessageFlagsEphemeral,
		Content: fmt.Sprintf("Took %d Tokens from <@%s>", amount, member.Id),
	})
	return err
}
