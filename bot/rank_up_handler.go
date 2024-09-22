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

		if tt.NextRank == 0 {
			continue
		}

		logger.WithField("member", member).WithField("count", count).WithField("nextRank", tt.NextRank).Debug("checking if member needs rank up")

		needsRankUp = append(needsRankUp, tt)
	}

	logger.WithField("count", len(needsRankUp)).Debug("need to rank up")

	if len(needsRankUp) == 0 {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "There are no members that need a rank up",
		})
		return nil
	}

	// output the list of members that need to be ranked up
	logger.WithField("members", needsRankUp).Debug("need to rank up")

	embed := &discordgo.MessageEmbed{
		Title:  "Members that need a Rank Up",
		Fields: []*discordgo.MessageEmbedField{},
	}
	for _, member := range needsRankUp {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "",
			Value: fmt.Sprintf("<@%s> to %s (%d Events)", member.Member.Id, member.NextRank.String(), member.Count),
		})
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
