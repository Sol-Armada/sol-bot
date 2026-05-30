package settingshandler

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/members"
)

func dmOptOutButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	id := i.User.ID
	if i.Member != nil {
		id = i.Member.User.ID
	}
	member, err := members.Get(id)
	if err != nil {
		return err
	}

	if member.DmOptOut {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredMessageUpdate,
		})
	}

	if err := member.OptOutOfDMs(); err != nil {
		return err
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "You have successfully opted out of direct messages. You will no longer receive direct messages from the bot. Use `/profile` in SolArmada to opt back in if you change your mind.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
