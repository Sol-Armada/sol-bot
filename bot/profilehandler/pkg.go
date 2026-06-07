package profilehandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	attdnc "github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/bot/internal/command"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/ranks"
	"github.com/sol-armada/sol-bot/rsi"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/tokens"
	"github.com/sol-armada/sol-bot/utils"
	"golang.org/x/exp/slices"
)

type ProfileCommand struct{}

var _ command.ApplicationCommand = (*ProfileCommand)(nil)

func New() command.ApplicationCommand {
	return &ProfileCommand{}
}

// AutocompleteHandler implements [command.ApplicationCommand].
func (c *ProfileCommand) AutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// ButtonHandler implements [command.ApplicationCommand].
func (c *ProfileCommand) ButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)

	customId := i.MessageComponentData().CustomID
	logger.Debug("handling profile command button interaction", "custom_id", customId)

	switch customId {
	case "profile:opt-out":
		member := utils.GetMemberFromContext(ctx).(*members.Member)
		member.DmOptOut = true
		if err := member.Save(); err != nil {
			return errors.Wrap(err, "saving member after opting out of DMs")
		}
	case "profile:opt-in":
		member := utils.GetMemberFromContext(ctx).(*members.Member)
		member.DmOptOut = false
		if err := member.Save(); err != nil {
			return errors.Wrap(err, "saving member after opting in to DMs")
		}
	}

	attendedEventCount, err := attdnc.GetMemberAttendanceCount(i.Member.User.ID)
	if err != nil {
		return errors.Wrap(err, "getting member attendance count")
	}

	profileMessage := profileMessage(ctx, utils.GetMemberFromContext(ctx).(*members.Member), attendedEventCount, true)

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     profileMessage.Embeds,
			Components: profileMessage.Components,
		},
	})
}

// CommandHandler implements [command.ApplicationCommand].
func (c *ProfileCommand) CommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)

	member := utils.GetMemberFromContext(ctx).(*members.Member)

	msg, err := s.InteractionResponse(i.Interaction)
	if err != nil {
		return err
	}

	if member.Name == "" {
		logger.Debug("no member found")

		if _, err := s.FollowupMessageEdit(i.Interaction, msg.ID, &discordgo.WebhookEdit{
			Content: new("You have not been onboarded yet! Contact an @Officer for some help!"),
		}); err != nil {
			return errors.Wrap(err, "responding to attendance command interaction: no member found")
		}

		return nil
	}

	data := i.ApplicationCommandData()

	if len(data.Options) > 0 && member.IsOfficer() {
		logger.Debug("getting profile of other member")
		otherMemberId := func() string {
			v := getOptionValue(data.Options, "member")
			if v != nil {
				return v.UserValue(s).ID
			}
			return ""
		}()

		if otherMemberId != "" {
			var err error
			member, err = members.Get(otherMemberId)
			if err != nil {
				if !errors.Is(err, members.MemberNotFound) {
					return errors.Wrap(err, "getting member for profile command")
				}

				discordMember, err := s.GuildMember(i.GuildID, otherMemberId)
				if err != nil {
					return errors.Wrap(err, "creating new guild member")
				}
				member = members.New(discordMember)
			}
		}

		forceOption := data.GetOption("force_update")
		if forceOption != nil && forceOption.BoolValue() { // update the member before getting their profile
			logger.Debug("force updating member")

			guildMember, err := s.GuildMember(i.GuildID, member.Id)
			if err != nil {
				return errors.Wrap(err, "getting guild member")
			}

			if err := member.UpdateFromDiscordMember(guildMember); err != nil {
				return errors.Wrap(err, "updating member from discord member")
			}

			member.Name = member.GetTrueNick(guildMember)

			profile, err := rsi.GetRSIInfo(member.Name)
			if err != nil {
				if errors.Is(err, rsi.ErrForbidden) {
					return err
				}

				if errors.Is(err, context.DeadlineExceeded) {
					return context.DeadlineExceeded
				}

				if errors.Is(err, customerrors.RsiDown) {
					_, err := s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
						Content: "RSI looks to be down. Unable to force a profile update. Please try again later.",
					})
					return err
				}

				if !errors.Is(err, rsi.ErrUserNotFound) {
					return errors.Wrap(err, "getting rsi info")
				}

				logger.Debug("rsi user not found", "member", member, "error", err.Error())
			} else {
				affiliations := make([]string, len(profile.Affiliation))
				for i, aff := range profile.Affiliation {
					affiliations[i] = aff.Name
				}

				if err := member.UpdateRsiInfo(); err != nil {
					return errors.Wrap(err, "updating member RSI info")
				}

				if err := member.Save(); err != nil {
					return errors.Wrap(err, "saving member after applying RSI profile")
				}
			}

			discordMember, err := s.GuildMember(i.GuildID, member.Id)
			if err != nil {
				return errors.Wrap(err, "getting discord member")
			}

			if slices.Contains(discordMember.Roles, settings.GetString("DISCORD.ROLE_IDS.RECRUIT")) {
				logger.Debug("is recruit")
				member.Rank = ranks.Recruit
				member.IsAffiliate = false
				member.IsAlly = false
			}
			if slices.Contains(discordMember.Roles, settings.GetString("DISCORD.ROLE_IDS.ALLY")) {
				logger.Debug("is ally")
				member.Rank = ranks.None
				member.IsAffiliate = false
				member.IsAlly = true
			}
			if discordMember.User.Bot {
				logger.Debug("is bot")
				member.Rank = ranks.None
				member.IsAffiliate = false
				member.IsAlly = false
				member.IsBot = true
			}

			if err := member.Save(); err != nil {
				return err
			}
		}
	}

	attendedEventCount, err := attdnc.GetMemberAttendanceCount(member.Id)
	if err != nil {
		return errors.Wrap(err, "getting member attendance count")
	}

	profileMessage := profileMessage(ctx, member, attendedEventCount, member.Id == i.Member.User.ID)

	params := &discordgo.WebhookEdit{
		Content:    new(""),
		Embeds:     &profileMessage.Embeds,
		Components: &profileMessage.Components,
	}

	if _, err := s.FollowupMessageEdit(i.Interaction, msg.ID, params); err != nil {
		return errors.Wrap(err, "editing followup message")
	}

	return nil
}

// ModalHandler implements [command.ApplicationCommand].
func (c *ProfileCommand) ModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// Name implements [command.ApplicationCommand].
func (c *ProfileCommand) Name() string {
	return "profile"
}

// OnAfter implements [command.ApplicationCommand].
func (c *ProfileCommand) OnAfter(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// OnBefore implements [command.ApplicationCommand].
func (c *ProfileCommand) OnBefore(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// OnError implements [command.ApplicationCommand].
func (c *ProfileCommand) OnError(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Error("handling profile command", "error", err)
}

// SelectMenuHandler implements [command.ApplicationCommand].
func (c *ProfileCommand) SelectMenuHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

// Setup implements [command.ApplicationCommand].
func (c *ProfileCommand) Setup() (*discordgo.ApplicationCommand, error) {
	return &discordgo.ApplicationCommand{
		Name:        "profile",
		Description: "View your profile in Sol Armada",
		Type:        discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "member",
				Description: "The member to view the profile of (officers only)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "force_update",
				Description: "Whether to force update the member's information before viewing the profile (officers only)",
				Required:    false,
			},
		},
	}, nil
}

func (c *ProfileCommand) SetupAliases() ([]*discordgo.ApplicationCommand, error) {
	return nil, nil
}

func getOptionValue(options []*discordgo.ApplicationCommandInteractionDataOption, name string) *discordgo.ApplicationCommandInteractionDataOption {
	for _, option := range options {
		if option.Name == name {
			return option
		}
	}
	return nil
}

func profileMessage(ctx context.Context, member *members.Member, attendedEventCount int, isMember bool) *discordgo.Message {
	logger := utils.GetLoggerFromContext(ctx)

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

	if member.Rank == ranks.Guest {
		emFields[1].Value = "Guest"
	}

	validated := "No"
	if member.ValidatedAt != nil && !member.ValidatedAt.IsZero() {
		validated = "Yes"
	}
	if member.OnRsi() {
		po := member.RsiInfo.PrimaryOrg
		if po == "" {
			po = "None set"
		}
		rsiFields := []*discordgo.MessageEmbedField{
			{
				Name:   "RSI Profile URL",
				Value:  rsi.UserProfileURL(member.Name),
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
		logger.Error("getting balance", "error", err)
		balance = 0
	}

	emFields = append(emFields, &discordgo.MessageEmbedField{
		Name: "Tokens",
		Value: func() string {
			if balance == 0 && err != nil {
				return "There was an issue"
			}
			return fmt.Sprintf("%d", balance)
		}(),
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

	components := []discordgo.MessageComponent{
		func() discordgo.Button {
			if member.DmOptOut {
				return discordgo.Button{
					Label:    "Opt-In to Bot DMs",
					CustomID: "profile:opt-in",
					Emoji: &discordgo.ComponentEmoji{
						Name: "✅",
					},
				}
			}
			return discordgo.Button{
				Label:    "Opt-Out of Bot DMs",
				CustomID: "profile:opt-out",
				Emoji: &discordgo.ComponentEmoji{
					Name: "🚫",
				},
			}
		}(),
	}

	if (member.ValidatedAt == nil || member.ValidatedAt.IsZero()) && isMember {
		components = append(components,
			discordgo.Button{
				Label:    "Validate RSI Profile",
				CustomID: fmt.Sprintf("validate:start:%s", member.Id),
				Emoji: &discordgo.ComponentEmoji{
					Name: "✅",
				},
			},
		)
	}

	return &discordgo.Message{
		Embeds: []*discordgo.MessageEmbed{em},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: components,
			},
		},
	}
}
