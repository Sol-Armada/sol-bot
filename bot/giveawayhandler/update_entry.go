package giveawayhandler

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/giveaway"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/utils"
)

func updateEntry(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	giveawayId := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	entries := i.MessageComponentData().Values

	member := utils.GetMemberFromContext(ctx).(*members.Member)

	g := giveaway.GetGiveaway(giveawayId)

	if !g.CanParticipate(member.Id) {
		customerrors.ErrorResponse(s, i.Interaction, "You did not attend this event! You don't qualify for this giveaway.", nil)
		return nil
	}

	g.AddMemberToItems(entries, member.Id)

	_ = g.UpdateMessage()

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "",
			Embeds:  []*discordgo.MessageEmbed{g.GetViewEntriesEmbed(member.Id)},
		},
	})
}
