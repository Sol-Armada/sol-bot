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

	g := giveaway.GetGiveaway(giveawayId)

	a, err := attendance.Get(g.AttendanceId)
	if err != nil {
		if err == attendance.ErrAttendanceNotFound {
			customerrors.ErrorResponse(s, i.Interaction, "Attendance record not found", nil)
			return nil
		}
		return err
	}

	if !a.HasMember(member.Id, true) {
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
