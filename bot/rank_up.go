package bot

import (
	"context"

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
		count, err := attendance.GetMemberAttendanceCount(member.Id)
		if err != nil {
			return err
		}

		if !member.IsRanked() || member.Rank == ranks.Recruit {
			continue
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

		needsRankUp = append(needsRankUp, tt)
	}

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
			Value: "<@" + member.Member.Id + "> to " + member.NextRank.String(),
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
