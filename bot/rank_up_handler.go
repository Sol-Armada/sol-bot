package bot

import (
	"context"
	"fmt"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/ranks"
	"github.com/sol-armada/sol-bot/utils"
)

func rankUpsCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("rank ups command handler")

	if !allowed(i.Member, "ATTENDANCE") {
		return InvalidPermissions
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	// get members
	membersList, err := members.List(0)
	if err != nil {
		return err
	}

	// check if any members need to rank up
	type t struct {
		Member   members.Member
		NextRank ranks.Rank
		Count    int
	}
	needsRankUp := []t{}
	for _, member := range membersList {
		if !member.IsRanked() || member.IsGuest || member.IsAlly || member.IsAffiliate {
			continue
		}

		logger.WithField("member", member.Id).Debug("checking if member needs rank up")

		count, err := attendance.GetMemberAttendanceCount(member.Id)
		if err != nil {
			return err
		}

		tt := t{Member: member, Count: count}

		if member.Rank == ranks.Recruit && count >= 3 {
			tt.NextRank = ranks.Member
		}
		if member.Rank == ranks.Member && count >= 10 {
			tt.NextRank = ranks.Technician
		}
		if member.Rank == ranks.Technician && count >= 20 {
			tt.NextRank = ranks.Specialist
		}

		if tt.NextRank != 0 {
			needsRankUp = append(needsRankUp, tt)
		}
	}

	if len(needsRankUp) == 0 {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "There are no members that need a rank up",
		})
		return nil
	}

	// output the list of members that need to be ranked up
	logger.WithField("members", needsRankUp).Debug("need to rank up")

	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Members to Rank Up",
			Value:  "",
			Inline: true,
		},
	}
	embed := &discordgo.MessageEmbed{
		Title:  "",
		Fields: fields,
	}

	ind := 0
	for _, member := range needsRankUp {
		if member.NextRank == 0 {
			continue
		}

		if ind%10 == 0 && ind != 0 {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "Members to Rank Up (continued)",
				Value:  "",
				Inline: true,
			})
		}

		field := fields[len(fields)-1]
		field.Value += fmt.Sprintf("<@%s> to %s (%d Events)", member.Member.Id, member.NextRank.String(), member.Count)

		// if not the 10th member, add a newline
		if ind%10 != 9 {
			field.Value += "\n"
		}

		ind++
	}

	// create followup
	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "These members need a rank up",
		Embeds:  []*discordgo.MessageEmbed{embed},
	})
	if err != nil {
		return err
	}

	return nil
}
