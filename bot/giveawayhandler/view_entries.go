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

func viewEntries(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	member := utils.GetMemberFromContext(ctx).(*members.Member)

	giveawayId := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	g := giveaway.GetGiveaway(giveawayId)

	if g == nil {
		return customerrors.InvalidGiveaway
	}

	a, err := attendance.Get(g.AttendanceId)
	if err != nil {
		a = &attendance.Attendance{
			Name: "",
		}
	}

	if !a.HasMember(member.Id, true) {
		customerrors.ErrorResponse(s, i.Interaction, "You did not attend this event! You don't qualify for this giveaway.", nil)
		return nil
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "",
			Embeds:  []*discordgo.MessageEmbed{g.GetViewEntriesEmbed(member.Id)},
		},
	})
}
