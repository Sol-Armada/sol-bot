package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	attdnc "github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/ranks"
	"github.com/sol-armada/sol-bot/rsi"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/utils"
	"golang.org/x/exp/slices"
)

func profileCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	member := utils.GetMemberFromContext(ctx).(*members.Member)

	if member.Name == "" {
		logger.Debug("no user found")

		// if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		// 	Type: discordgo.InteractionResponseChannelMessageWithSource,
		// 	Data: &discordgo.InteractionResponseData{
		// 		Content: "You have not been onboarded yet! Contact an @Officer for some help!",
		// 		Flags:   discordgo.MessageFlagsEphemeral,
		// 	},
		// }); err != nil {
		// 	return errors.Wrap(err, "responding to attendance command interaction: no user found")
		// }

		if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "You have not been onboarded yet! Contact an @Officer for some help!",
		}); err != nil {
			return errors.Wrap(err, "responding to attendance command interaction: no user found")
		}

		return nil
	}

	data := i.ApplicationCommandData()

	if len(data.Options) > 0 {
		if member.Rank > ranks.Lieutenant {
			return InvalidPermissions
		}

		otherMemberId := data.Options[0].UserValue(s).ID

		if otherMemberId != "" {
			otherMember, err := members.Get(otherMemberId)
			if err != nil {
				if !errors.Is(err, members.MemberNotFound) {
					return errors.Wrap(err, "getting member for profile command")
				}

				// if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				// 	Type: discordgo.InteractionResponseChannelMessageWithSource,
				// 	Data: &discordgo.InteractionResponseData{
				// 		Content: "That user was not found in the system!",
				// 		Flags:   discordgo.MessageFlagsEphemeral,
				// 	},
				// }); err != nil {
				// 	return errors.Wrap(err, "responding to attendance command interaction: no user found")
				// }

				if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: "That user was not found in the system!",
				}); err != nil {
					return errors.Wrap(err, "responding to attendance command interaction: no user found")
				}
			}

			if len(data.Options) > 1 && data.Options[1].BoolValue() { // update the member before getting their profile
				if err := rsi.UpdateRsiInfo(otherMember); err != nil {
					if strings.Contains(err.Error(), "Forbidden") || strings.Contains(err.Error(), "Bad Gateway") {
						return err
					}

					if strings.Contains(err.Error(), "context deadline exceeded") {
						return context.DeadlineExceeded
					}

					if !errors.Is(err, rsi.RsiUserNotFound) {
						return errors.Wrap(err, "getting rsi info")
					}

					logger.WithField("member", member).Debug("rsi user not found")
					otherMember.RSIMember = false
				}

				discordMember, err := s.GuildMember(i.GuildID, otherMember.Id)
				if err != nil {
					return errors.Wrap(err, "getting discord member")
				}

				if slices.Contains(discordMember.Roles, settings.GetString("DISCORD.ROLE_IDS.RECRUIT")) {
					logger.Debug("is recruit")
					member.Rank = ranks.Recruit
					member.IsAffiliate = false
					member.IsAlly = false
					member.IsGuest = false
				}
				if discordMember.User.Bot {
					member.IsBot = true
				}

				if err := otherMember.Save(); err != nil {
					return err
				}
			}

			member = otherMember
		}
	}

	attendedEventCount, err := attdnc.GetMemberAttendanceCount(member.Id)
	if err != nil {
		return errors.Wrap(err, "getting member attendance count")
	}

	emFields := []*discordgo.MessageEmbedField{
		{
			Name:   "RSI Handle",
			Value:  member.Name,
			Inline: false,
		},
		{
			Name:   "Rank",
			Value:  member.Rank.String(),
			Inline: true,
		},
		{
			Name:   "Event Attendance Count",
			Value:  fmt.Sprintf("%d", attendedEventCount),
			Inline: true,
		},
	}

	validated := "No"
	if member.Validated {
		validated = "Yes"
	}
	if member.RSIMember {
		rsiFields := []*discordgo.MessageEmbedField{
			{
				Name:   "RSI Profile URL",
				Value:  fmt.Sprintf("https://robertsspaceindustries.com/citizens/%s", member.Name),
				Inline: false,
			},
			{
				Name:   "RSI Validated (/validate)",
				Value:  validated,
				Inline: false,
			},
		}
		emFields = append(emFields, rsiFields...)
	}

	memberIssues := attdnc.Issues(member)
	if len(memberIssues) > 0 {
		emFields = append(emFields, &discordgo.MessageEmbedField{
			Name:   "Restrictions to Promotion",
			Value:  strings.Join(memberIssues, ", "),
			Inline: false,
		})
	}

	em := &discordgo.MessageEmbed{
		Title:       "Profile",
		Description: "Information about you in Sol Armada",
		Color:       0x00FFFF,
		Fields:      emFields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Last updated %s", member.Updated.UTC().Format("2006-01-02 15:04:05 MST")),
		},
	}

	// if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
	// 	Type: discordgo.InteractionResponseChannelMessageWithSource,
	// 	Data: &discordgo.InteractionResponseData{
	// 		Flags: discordgo.MessageFlagsEphemeral,
	// 		Embeds: []*discordgo.MessageEmbed{
	// 			em,
	// 		},
	// 	},
	// }); err != nil {
	// 	return errors.Wrap(err, "responding to attendance command interaction")
	// }

	if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "",
		Embeds:  []*discordgo.MessageEmbed{em},
	}); err != nil {
		return errors.Wrap(err, "responding to attendance command interaction")
	}

	return nil
}
