package bot

import (
	"context"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/rsi"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/utils"
)

func setupOnboarding() error {
	logger := log.WithField("func", "setupOnboarding")

	logger.Debug("setting up onboarding")

	msgs, err := bot.ChannelMessages(settings.GetString("FEATURES.ONBOARDING.INPUT_CHANNEL_ID"), 1, "", "", "")
	if err != nil {
		return err
	}

	content := `Welcome to Sol Armada!

Select a reason you joined below. We will ask a few questions and someone will be available in the <#223290459726807040> to verbally onboard you!`

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "A member recruited me",
					CustomID: "onboarding:choice:recruited",
					Style:    discordgo.PrimaryButton,
					Emoji:    &discordgo.ComponentEmoji{Name: "🤝"},
				},
				discordgo.Button{
					Label:    "Found Sol Armada on RSI",
					CustomID: "onboarding:choice:rsi",
					Style:    discordgo.PrimaryButton,
					Emoji:    &discordgo.ComponentEmoji{Name: "🔍"},
				},
				discordgo.Button{
					Label:    "Some other way",
					CustomID: "onboarding:choice:other",
					Style:    discordgo.PrimaryButton,
					Emoji:    &discordgo.ComponentEmoji{Name: "❔"},
				},
			},
		},
	}

	if len(msgs) == 0 {
		logger.Debug("no onboarding message found")
		if _, err := bot.ChannelMessageSendComplex(settings.GetString("FEATURES.ONBOARDING.INPUT_CHANNEL_ID"), &discordgo.MessageSend{
			Content:    content,
			Components: components,
		}); err != nil {
			return err
		}

		return nil
	}

	logger.Debug("onboarding message found")
	if _, err := bot.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    msgs[0].ChannelID,
		ID:         msgs[0].ID,
		Content:    &content,
		Components: &components,
	}); err != nil {
		return err
	}

	return nil
}

func onboardingButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)

	logger.Debug("onboarding button handler")

	member := utils.GetMemberFromContext(ctx).(*members.Member)

	if member.OnboardedAt != nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You have already been onboarded!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})

		return nil
	}

	data := i.MessageComponentData()
	choice := strings.Split(data.CustomID, ":")[2]

	questions := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID: "rsi_handle",
					Label:    "Your RSI Handle",
					Style:    discordgo.TextInputShort,
					Required: true,
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID:    "play_time",
					Label:       "How long have you been playing Star Citizen?",
					Style:       discordgo.TextInputShort,
					Placeholder: "Example: 2 years",
					Required:    true,
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID:    "gameplay",
					Label:       "What gamplay are you most interested in?",
					Style:       discordgo.TextInputShort,
					Placeholder: "Combat, Rescue, Mining, etc",
					Required:    true,
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID: "age",
					Label:    "How old are you?",
					Style:    discordgo.TextInputShort,
					Required: true,
				},
			},
		},
	}

	switch choice {
	case "recruited":
		questions = append(questions, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID:    "recruiter",
					Label:       "Who recruited you?",
					Style:       discordgo.TextInputShort,
					Placeholder: "The recruiter's RSI handle",
					Required:    true,
				},
			},
		})
	case "other":
		questions = append(questions, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID: "other",
					Label:    "How did you find us?",
					Style:    discordgo.TextInputParagraph,
					Required: true,
				},
			},
		})
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   "onboarding:onboard",
			Title:      "Onboarding",
			Components: questions,
		},
	}); err != nil {
		return err
	}

	return nil
}

func onboardingModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)

	member := utils.GetMemberFromContext(ctx).(*members.Member)

	logger = logger.WithField("member", member.Id)

	logger.Info("onboarding modal handler")

	data := i.ModalSubmitData()

	if err := deferInteraction(s, i); err != nil {
		return errors.Wrap(err, "responding")
	}

	rsiHandle := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	playTime := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	gameplay := data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	age := data.Components[3].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	recruiter := ""
	other := ""

	if len(data.Components) > 4 {
		if actionRow, ok := data.Components[4].(*discordgo.ActionsRow); ok {
			if comp, ok := actionRow.Components[0].(*discordgo.TextInput); ok {
				switch comp.CustomID {
				case "recruiter":
					recruiter = comp.Value
				case "other":
					other = comp.Value
				}
			}
		}
	}

	member.LegacyPlaytime = playTime
	member.LegacyGamplay = gameplay
	member.LegacyAge = age

	if recruiter != "" {
		member.LegacyRecruiter = recruiter
	}

	if other != "" {
		member.LegacyOther = other
	}

	if err := member.Save(); err != nil {
		return errors.Wrap(err, "onboarding modal handler: failed to save member first")
	}

	// validate rsi handle
	if !rsi.ValidHandle(rsiHandle) {
		logger.Debug("invalid RSI handle")

		if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "I couldn't find that RSI handle!\n\nPlease make sure it is correct and try again.\nYour RSI handle can be found on your public RSI profile page or in your settings here: https://robertsspaceindustries.com/account/settings",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Try Again",
							CustomID: "onboarding:tryagain:" + i.Interaction.Member.User.ID,
						},
					},
				},
			},
		}); err != nil {
			return errors.Wrap(err, "onboarding modal handler: responding to invalid rsi handle")
		}

		return nil
	}

	member.Name = rsiHandle

	if err := member.Save(); err != nil {
		return errors.Wrap(err, "onboarding modal handler: saving member second")
	}

	ctx = utils.SetMemberToContext(ctx, member)

	return finishOnboarding(ctx, s, i)
}

func onboardingTryAgainHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)

	logger.Debug("onboarding try again handler")

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "onboarding:rsihandle:" + i.Interaction.Member.User.ID,
			Title:    "What is your RSI Handle?",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "rsi_handle",
							Label:       "Your RSI handle",
							Style:       discordgo.TextInputShort,
							Placeholder: "Your handle can be found on your public RSI page",
							Required:    true,
						},
					},
				},
			},
		},
	}); err != nil {
		return errors.Wrap(err, "onboarding try again handler")
	}

	return nil
}

func onboardingTryAgainModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)

	member := utils.GetMemberFromContext(ctx).(*members.Member)

	logger.WithField("member", member.Id)

	logger.Info("onboarding try again modal handler")

	data := i.ModalSubmitData()
	rsiHandle := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	if err := deferInteraction(s, i); err != nil {
		return errors.Wrap(err, "responding")
	}

	if !rsi.ValidHandle(rsiHandle) {
		logger.Debug("invalid RSI handle")

		if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "I couldn't find that RSI handle!\n\nPlease make sure it is correct and try again.\nYour RSI handle can be found on your public RSI profile page or in your settings here: https://robertsspaceindustries.com/account/settings",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Try Again",
							CustomID: "onboarding:tryagain:" + i.Interaction.Member.User.ID,
						},
					},
				},
			},
		}); err != nil {
			return errors.Wrap(err, "onboarding modal handler: responding to invalid rsi handle")
		}
	}

	member.Name = rsiHandle

	if err := member.Save(); err != nil {
		return errors.Wrap(err, "onboarding try again modal handler: saving member")
	}

	ctx = utils.SetMemberToContext(ctx, member)

	return finishOnboarding(ctx, s, i)
}

func finishOnboarding(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)

	member := utils.GetMemberFromContext(ctx).(*members.Member)

	logger.WithField("member", member.Id)

	logger.Info("finishing onboarding")

	if _, err := s.GuildMemberEdit(bot.GuildId, member.Id, &discordgo.GuildMemberParams{
		Nick: member.Name,
	}); err != nil {
		return err
	}

	if _, err := s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Content: "Thank you for answering our questions! Your Discord nickname has been set to your RSI handle. You can contact someone in <#223290459726807040> to get verbally onboarded!",
		Flags:   discordgo.MessageFlagsEphemeral,
	}); err != nil {
		return errors.Wrap(err, "finishing onboarding: responding")
	}

	fields := []*discordgo.MessageEmbedField{
		{Name: "RSI Profile", Value: "https://robertsspaceindustries.com/citizens/" + member.Name},
		{Name: "Primary Org", Value: "https://robertsspaceindustries.com/orgs/" + member.PrimaryOrg},
		{Name: "Affiliate Orgs", Value: strings.Join(member.Affilations, ", ")},
		{Name: "Playtime", Value: member.LegacyPlaytime},
		{Name: "Gameplay", Value: member.LegacyGamplay},
		{Name: "Age", Value: member.LegacyAge},
	}

	if member.LegacyRecruiter != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Recruiter", Value: member.LegacyRecruiter})
	}

	if member.LegacyOther != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "How they found us", Value: member.LegacyOther})
	}

	if _, err := s.ChannelMessageSendComplex(settings.GetString("FEATURES.ONBOARDING.OUTPUT_CHANNEL_ID"), &discordgo.MessageSend{
		Content: "Onboarding information for " + i.Member.Mention(),
		Embeds: []*discordgo.MessageEmbed{
			{
				Fields:    fields,
				Timestamp: member.Joined.Format(time.RFC3339),
			},
		},
	}); err != nil {
		return errors.Wrap(err, "finishing onboarding: sending onboarded message")
	}

	return nil
}

func deferInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}
