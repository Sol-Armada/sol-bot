package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/rsi"
	"github.com/sol-armada/sol-bot/utils"
)

func validateCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {

	member, err := members.Get(i.Member.User.ID)
	if err != nil {
		return err
	}

	if member.Validated {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You are already validated! Congrats!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		}); err != nil {
			return err
		}

		return nil
	}

	code := utils.GenerateRandomAlphaNumeric(8)
	member.ValidationCode = code
	if err := member.Save(); err != nil {
		return err
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: fmt.Sprintf("Please insert this generated code into the Short Bio section of your [RSI profile](https://robertsspaceindustries.com/account/profile), then click \"APPLY ALL CHANGES\" on the page. Wait 5 seconds, then click the \"Validate\" button.\n\n%s", code),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Validate",
							CustomID: fmt.Sprintf("onboarding:validate:%s", i.Member.User.ID),
							Emoji:    &discordgo.ComponentEmoji{Name: "✅"},
						},
					},
				},
			},
		},
	}); err != nil {
		return err
	}

	return nil
}

func validateButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	memberId := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	member, err := members.Get(memberId)
	if err != nil {
		return err
	}

	if member.ValidationCode == "" {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "It looks like I am not prepared to validate your profile. Please try the command again or let the @Officers know if you keep running into this message.",
			},
		}); err != nil {
			return err
		}

		return nil
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	waitTime := time.Duration(2 * time.Second)
	ticker := time.NewTicker(waitTime)

	attempt := 1
	for {
		bio, err := rsi.GetBio(member.GetTrueNick(i.Member))
		if err != nil {
			return err
		}

		if !strings.Contains(bio, member.ValidationCode) {
			if attempt == 3 {
				_, _ = s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
					Flags:   discordgo.MessageFlagsEphemeral,
					Content: "I could not find the code on your profile. Please try the command again and give a minute after adding the code to your short bio before clicking \"Validate\"",
				})
				ticker.Stop()
				return nil
			}

			goto CONTINUE
		}

		ticker.Stop()
		break

	CONTINUE:
		attempt++
		ticker.Reset(waitTime)
		<-ticker.C
	}

	member.Validated = true
	member.ValidationCode = ""
	if err := member.Save(); err != nil {
		return err
	}

	if _, err := s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Flags:   discordgo.MessageFlagsEphemeral,
		Content: "Your account has been validated! You can remove the code from your bio.",
	}); err != nil {
		return err
	}

	return nil
}
