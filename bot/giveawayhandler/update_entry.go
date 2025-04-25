package giveawayhandler

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/giveaway"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/utils"
)

func updateEntry(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	giveawayId := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	entries := i.MessageComponentData().Values

	member := utils.GetMemberFromContext(ctx).(*members.Member)

	g := giveaway.GetGiveaway(giveawayId).AddMemberToItems(entries, member)

	attendance, err := attendance.Get(g.Attendance.Id)
	if err != nil {
		return err
	}

	if !attendance.HasMember(member.Id, true) {
		customerrors.ErrorResponse(s, i.Interaction, "You did not attend this event! You don't qualify for this giveaway.", nil)
		return nil
	}

	components := g.GetComponents()
	embeds := g.GetEmbed()

	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    g.ChannelId,
		ID:         g.MessageId,
		Components: &components,
		Embeds:     &[]*discordgo.MessageEmbed{embeds},
	}); err != nil {
		return err
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "",
			Embeds:  []*discordgo.MessageEmbed{g.GetViewEntriesEmbed(member)},
		},
	})
}
