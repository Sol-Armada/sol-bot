package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	attdnc "github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/members"
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
			Value:  fmt.Sprintf("%d", member.LegacyEvents),
			Inline: true,
		},
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
				Value:  strconv.FormatBool(member.Validated),
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
			Text: fmt.Sprintf("Last updated <t:%d:R>", member.Updated.Unix()),
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
