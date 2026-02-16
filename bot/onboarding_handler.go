package bot

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/rsi"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/utils"
)

func setupOnboarding() error {
	logger := slog.Default()

	logger.Debug("setting up onboarding")

	logo := &discordgo.File{
		Name:        "logo.png",
		ContentType: "image/png",
	}

	logoPath := settings.GetStringWithDefault("LOGO", "")

	var file []byte
	var files []*discordgo.File
	var err error

	if logoPath != "" {
		file, err = os.ReadFile(logoPath)
		if err != nil {
			logoPath = ""
			logger.Warn("failed to read logo file, continuing without logo", "error", err)
			goto CONTINUE
		}

		// create reader from bytes
		logo.Reader = strings.NewReader(string(file))
		if logoPath != "" {
			files = []*discordgo.File{logo}
		}
	} else {
		logger.Debug("no logo found")
	}

CONTINUE:

	content := `# Welcome to Sol Armada!

Let us know how you found us by clicking one of the buttons below. This will help us improve our recruitment efforts and better understand our community.
If you run into any issues, please reach out in <#223290459726807040> for assistance.

## If you are here to join Sol Armada
After applying to the Org on RSI and paticipating in a brief verbal onboarding, with an <@&398414253171671041> or <@&1109958022362562672>, we will mark you as a Recruit, granting access to more channels here on Discord. We don't accept applications until you attend 3 official Org events.

## If you are an ambassador or content creator
Let's chat! Message <@91622043040124928>, Sol Armada's Diplomat and Admiral.

— Sol Armada Org Administration

### [Sol Armada Handbook - Please read!](https://www.solarmada.space/fullhandbook)

### [Join the Org!](https://www.solarmada.space/new-recruits)
`

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

	msg := &discordgo.MessageSend{
		Content:    content,
		Components: components,
		Flags:      discordgo.MessageFlagsSuppressEmbeds,
	}

	if logoPath != "" {
		msg.Files = files
	}

	msgs, err := bot.ChannelMessages(settings.GetString("FEATURES.ONBOARDING.INPUT_CHANNEL_ID"), 1, "", "", "")
	if err != nil {
		return err
	}

	if len(msgs) == 0 {
		logger.Debug("no onboarding message found")
		if _, err := bot.ChannelMessageSendComplex(settings.GetString("FEATURES.ONBOARDING.INPUT_CHANNEL_ID"), msg); err != nil {
			return err
		}

		return nil
	}

	logger.Debug("onboarding message found")

	msgEdit := &discordgo.MessageEdit{
		Channel:    msgs[0].ChannelID,
		ID:         msgs[0].ID,
		Content:    &msg.Content,
		Components: &msg.Components,
		Flags:      discordgo.MessageFlagsSuppressEmbeds,
	}

	if _, err := bot.ChannelMessageEditComplex(msgEdit); err != nil {
		return err
	}

	return nil
}

func onboardingButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)

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
					Required: new(true),
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
					Required:    new(true),
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID:    "gameplay",
					Label:       "What gameplay are you most interested in?",
					Style:       discordgo.TextInputShort,
					Placeholder: "Combat, Rescue, Mining, etc",
					Required:    new(true),
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID: "age",
					Label:    "How old are you?",
					Style:    discordgo.TextInputShort,
					Required: new(true),
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
					Required:    new(true),
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
					Required: new(true),
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
	logger := utils.GetLoggerFromContext(ctx)

	member := utils.GetMemberFromContext(ctx).(*members.Member)

	logger = logger.With("member", member.Id)

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
	member.LegacyGameplay = gameplay
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
	logger := utils.GetLoggerFromContext(ctx)

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
							Required:    new(true),
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
	logger := utils.GetLoggerFromContext(ctx)

	member := utils.GetMemberFromContext(ctx).(*members.Member)

	logger.With("member", member.Id)

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
	logger := utils.GetLoggerFromContext(ctx)

	member := utils.GetMemberFromContext(ctx).(*members.Member)

	logger.With("member", member.Id)

	logger.Info("finishing onboarding")

	g, _ := s.Guild(bot.GuildId)
	if g != nil && i.Member.User.ID == g.OwnerID {
		goto SKIP
	}

	if _, err := s.GuildMemberEdit(bot.GuildId, member.Id, &discordgo.GuildMemberParams{
		Nick: member.Name,
	}); err != nil {
		return err
	}

SKIP:
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
		{Name: "Gameplay", Value: member.LegacyGameplay},
		{Name: "Age", Value: member.LegacyAge},
	}

	if member.LegacyRecruiter != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Recruiter", Value: member.LegacyRecruiter})
	}

	if member.LegacyOther != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "How they found us", Value: member.LegacyOther})
	}

	if member.MessageId != "" {
		if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Channel: member.ChannelId,
			ID:      member.MessageId,
			Embeds: &[]*discordgo.MessageEmbed{
				{
					Fields:    fields,
					Timestamp: member.Joined.Format(time.RFC3339),
				},
			},
		}); err != nil {
			goto CREATE
		}

		return nil
	}

CREATE:
	msg, err := s.ChannelMessageSendComplex(settings.GetString("FEATURES.ONBOARDING.OUTPUT_CHANNEL_ID"), &discordgo.MessageSend{
		Content: "Onboarding information for " + i.Member.Mention(),
		Embeds: []*discordgo.MessageEmbed{
			{
				Fields:    fields,
				Timestamp: member.Joined.Format(time.RFC3339),
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "finishing onboarding: sending onboarded message")
	}

	member.MessageId = msg.ID
	member.ChannelId = msg.ChannelID
	member.OnboardedAt = &msg.Timestamp

	return member.Save()
}

func deferInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}
