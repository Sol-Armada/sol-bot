package attendancehandler

import (
	"context"
	"fmt"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/pkg/errors"
	"github.com/rs/xid"
	attdnc "github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/config"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/utils"
	"go.mongodb.org/mongo-driver/bson"
)

func createCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("create attendance command")

	commandMember := utils.GetMemberFromContext(ctx).(*members.Member)

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	data := i.Interaction.ApplicationCommandData().Options[0]

	eventName := data.Options[0].StringValue()

	valid, err := config.ValidAttendanceName(eventName)
	if err != nil {
		return errors.Wrap(err, "checking if attendance name is valid")
	}
	if !valid {
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "That is not a valid attendance name! Please choose from the list given when creating a new attendance record.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return err
	}

	exists := false
	if _, err := xid.FromString(eventName); err == nil {
		exists = true
	}

	attendance := attdnc.New(eventName, commandMember)

	options := data.Options[1:]
	for _, option := range options {
		if option.Name == "tokens" {
			attendance.Tokenable = option.BoolValue()
			break
		}

		if option.Type == discordgo.ApplicationCommandOptionUser {
			member, err := members.Get(option.UserValue(s).ID)
			if err != nil {
				if !errors.Is(err, members.MemberNotFound) {
					return errors.Wrap(err, "getting member for new attendance")
				}

				if member == nil {
					attendance.WithIssues = append(attendance.WithIssues, &members.Member{
						Id:   option.UserValue(s).ID,
						Name: option.UserValue(s).Username,
					})
				}

				attendance.WithIssues = append(attendance.WithIssues, member)

				continue
			}

			attendance.AddMember(member)
		}
	}

	// save now incase there is an error with creating the message
	if err := attendance.Save(); err != nil {
		return errors.Wrap(err, "saving attendance record")
	}

	attandanceMessage := attendance.ToDiscordMessage()
	message, err := s.ChannelMessageSendComplex(attendance.ChannelId, attandanceMessage)
	if err != nil {
		return errors.Wrap(err, "sending attendance message")
	}
	attendance.MessageId = message.ID

	if err := attendance.Save(); err != nil {
		return errors.Wrap(err, "saving attendance record")
	}

	content := fmt.Sprintf("Attendance record https://discord.com/channels/%s/%s/%s created", i.GuildID, settings.GetString("FEATURES.ATTENDANCE.CHANNEL_ID"), attendance.MessageId)
	if exists {
		content = fmt.Sprintf("Attendance record https://discord.com/channels/%s/%s/%s updated", i.GuildID, settings.GetString("FEATURES.ATTENDANCE.CHANNEL_ID"), attendance.MessageId)
	}
	_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: content,
		Flags:   discordgo.MessageFlagsEphemeral,
	})

	return nil
}

func createAutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("attendance create autocomplete")

	names, err := config.GetAttendanceNames()
	if err != nil {
		return errors.Wrap(err, "getting names")
	}

	choices := []*discordgo.ApplicationCommandOptionChoice{}

	typed := i.Interaction.ApplicationCommandData().Options[0].Options[0].StringValue()
	matches := fuzzy.FindFold(typed, names)

	for _, name := range matches {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  name,
			Value: name,
		})
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
}

func TagAutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("attendance tag autocomplete")

	choices := []*discordgo.ApplicationCommandOptionChoice{}

	raw, err := config.GetConfig("attendance_tags")
	if err != nil {
		return errors.Wrap(err, "getting tags")
	}

	tags, ok := raw.(bson.A)
	if !ok {
		return errors.New("unable to parse tags")
	}

	for _, tag := range tags {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  tag.(string),
			Value: tag.(string),
		})
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	}); err != nil {
		return errors.Wrap(err, "responding to attendance tag auto complete")
	}

	return nil
}
