package bot

import (
	"context"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
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
				// discordgo.Button{
				// 	Label:    "Just visiting",
				// 	CustomID: "onboarding:choice:visiting",
				// 	Style:    discordgo.PrimaryButton,
				// 	Emoji:    &discordgo.ComponentEmoji{Name: "👋"},
				// },
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
		return nil
	}

	data := i.MessageComponentData()
	choice := strings.Split(data.CustomID, ":")[2]

	if choice == "visiting" {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Okay! If you change your mind and would like to join Sol Armada, please contact an @Officer in the airlock!",
			},
		}); err != nil {
			return err
		}

		return nil
	}

	questions := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID: "rsi_handle",
					Label:    "RSI Handle",
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

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   "onboarding:onboard",
			Title:      "Onboarding",
			Components: questions,
		},
	})

	return nil
}

func onboardingModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)

	logger.Debug("onboarding modal handler")

	member := utils.GetMemberFromContext(ctx).(*members.Member)

	if member.OnboardedAt != nil {
		return nil
	}

	data := i.ModalSubmitData()

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
		return err
	}

	// validate rsi handle
	if !rsi.ValidHandle(rsiHandle) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
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
			},
		}); err != nil {
			return err
		}
	}

	member.Name = rsiHandle

	if err := member.Save(); err != nil {
		return err
	}

	return finishOnboarding(ctx, s, i)
}

func onboardingTryAgainHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)

	logger.Debug("onboarding try again handler")

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "onboarding:rsihandle:" + i.Interaction.Member.User.ID,
			Title:    "Some questions about you",
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
		logger.WithError(err).Error("responding to choice")
	}

	return nil
}

func onboardingTryAgainModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)

	logger.Debug("onboarding try again modal handler")

	member := utils.GetMemberFromContext(ctx).(*members.Member)

	data := i.ModalSubmitData()
	rsiHandle := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	if !rsi.ValidHandle(rsiHandle) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "I couldn't find that RSI handle!\n\nPlease make sure it is correct and try again.\nYour RSI handle can be found on your public RSI profile page or in your settings here: https://robertsspaceindustries.com/account/settings",
				Flags:   discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.Button{
								Label:    "Try Again",
								CustomID: "onboarding:try_rsi_handle_again:" + i.Interaction.Member.User.ID,
							},
						},
					},
				},
			},
		}); err != nil {
			return err
		}
	}

	member.Name = rsiHandle

	if err := member.Save(); err != nil {
		return err
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Thank you for answering our questions! Your nickname has been set to your RSI handle. You can contact someone in the <#223290459726807040> to get verbally onboarded!",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		return err
	}

	return nil
}

func finishOnboarding(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)

	logger.Debug("finishing onboarding")

	member := utils.GetMemberFromContext(ctx).(*members.Member)

	if _, err := s.GuildMemberEdit(bot.GuildId, member.Id, &discordgo.GuildMemberParams{
		Nick: member.Name,
	}); err != nil {
		return err
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Thank you for answering our questions! Your nickname has been set to your RSI handle. You can contact someone in the <#223290459726807040> to get verbally onboarded!",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		return err
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
		return err
	}

	return nil
}
