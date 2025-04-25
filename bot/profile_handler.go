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
	"github.com/sol-armada/sol-bot/tokens"
	"github.com/sol-armada/sol-bot/utils"
	"golang.org/x/exp/slices"
)

func profileCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)

	if len(i.Member.Roles) == 0 {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "You are a guest! This command is not available to you",
			},
		})
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	member := utils.GetMemberFromContext(ctx).(*members.Member)

	if member.Name == "" {
		logger.Debug("no member found")

		if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "You have not been onboarded yet! Contact an @Officer for some help!",
		}); err != nil {
			return errors.Wrap(err, "responding to attendance command interaction: no member found")
		}

		return nil
	}

	data := i.ApplicationCommandData()

	if len(data.Options) > 1 && member.Rank <= ranks.Lieutenant {
		logger.Debug("getting profile of other member")
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

				discordMember, err := s.GuildMember(i.GuildID, otherMemberId)
				if err != nil {
					return errors.Wrap(err, "creating new guild member")
				}
				otherMember = members.New(discordMember)
			}

			if len(data.Options) > 1 && data.Options[1].BoolValue() { // update the member before getting their profile
				logger.Debug("force updating member")

				guildMember, err := s.GuildMember(i.GuildID, otherMember.Id)
				if err != nil {
					return errors.Wrap(err, "getting guild member")
				}

				otherMember.Name = otherMember.GetTrueNick(guildMember)

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

					logger.WithFields(log.Fields{"member": otherMember, "error": err.Error()}).Debug("rsi user not found")
					otherMember.RSIMember = false
				}

				discordMember, err := s.GuildMember(i.GuildID, otherMember.Id)
				if err != nil {
					return errors.Wrap(err, "getting discord member")
				}

				if slices.Contains(discordMember.Roles, settings.GetString("DISCORD.ROLE_IDS.RECRUIT")) {
					logger.Debug("is recruit")
					otherMember.Rank = ranks.Recruit
					otherMember.IsAffiliate = false
					otherMember.IsAlly = false
					otherMember.IsGuest = false
				}
				if slices.Contains(discordMember.Roles, settings.GetString("DISCORD.ROLE_IDS.ALLY")) {
					logger.Debug("is ally")
					otherMember.Rank = ranks.None
					otherMember.IsAffiliate = false
					otherMember.IsAlly = true
					otherMember.IsGuest = false
				}
				if discordMember.User.Bot {
					logger.Debug("is bot")
					otherMember.Rank = ranks.None
					otherMember.IsAffiliate = false
					otherMember.IsAlly = false
					otherMember.IsGuest = false
					otherMember.IsBot = true
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

	rank := member.Rank.String()
	if rank == "" {
		rank = "None"
	}

	emFields := []*discordgo.MessageEmbedField{
		{
			Name:   "RSI Handle",
			Value:  member.Name,
			Inline: false,
		},
		{
			Name:   "Rank",
			Value:  rank,
			Inline: true,
		},
		{
			Name:   "Event Attendance Count",
			Value:  fmt.Sprintf("%d", attendedEventCount),
			Inline: true,
		},
	}

	if member.IsAffiliate {
		emFields[1].Value = "Affiliate"
	}

	if member.IsAlly {
		emFields[1].Value = "Ally/Friend"
	}

	if member.IsGuest {
		emFields[1].Value = "Guest"
	}

	validated := "No"
	if member.Validated {
		validated = "Yes"
	}
	if member.RSIMember {
		po := member.PrimaryOrg
		if po == "" {
			po = "None set"
		}
		rsiFields := []*discordgo.MessageEmbedField{
			{
				Name:   "RSI Profile URL",
				Value:  fmt.Sprintf("https://robertsspaceindustries.com/citizens/%s", member.Name),
				Inline: false,
			},
			{
				Name:   "Primary Org",
				Value:  po,
				Inline: false,
			},
			{
				Name:   "RSI Validated",
				Value:  validated,
				Inline: false,
			},
		}
		emFields = append(emFields, rsiFields...)
	}

	balance, err := tokens.GetBalanceByMemberId(member.Id)
	if err != nil {
		logger.WithError(err).Error("getting balance")
		balance = 0
	}

	emFields = append(emFields, &discordgo.MessageEmbedField{
		Name:   "Tokens",
		Value:  fmt.Sprintf("%d", balance),
		Inline: false,
	})

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
		Description: fmt.Sprintf("Information about <@%s> in Sol Armada", member.Id),
		Color:       0x00FFFF,
		Fields:      emFields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Last updated %s", member.Updated.UTC().Format("2006-01-02 15:04:05 MST")),
		},
	}

	params := &discordgo.WebhookParams{
		Content: "",
		Embeds:  []*discordgo.MessageEmbed{em},
	}

	if !member.Validated && member.Id == i.Member.User.ID {
		params.Components = []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Validate",
						CustomID: fmt.Sprintf("validate:start:%s", i.Member.User.ID),
					},
				},
			},
		}
	}

	if _, err := s.FollowupMessageCreate(i.Interaction, true, params); err != nil {
		return errors.Wrap(err, "responding to attendance command interaction")
	}

	return nil
}
