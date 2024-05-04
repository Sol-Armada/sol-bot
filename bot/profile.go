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
	"github.com/sol-armada/sol-bot/utils"
)

func profileCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)

	member := utils.GetMemberFromContext(ctx).(*members.Member)

	if member.Name == "" {
		logger.Debug("no user found")

		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You have not been onboarded yet! Contact an @Officer for some help!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
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

				if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "That user was not found in the system!",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				}); err != nil {
					return errors.Wrap(err, "responding to attendance command interaction: no user found")
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

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
			Embeds: []*discordgo.MessageEmbed{
				em,
			},
		},
	}); err != nil {
		return errors.Wrap(err, "responding to attendance command interaction")
	}

	return nil
}
