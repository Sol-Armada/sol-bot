package attendancehandler

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	attdnc "github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/utils"
)

func addMembersCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("add member attendance command")

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	data := i.Interaction.ApplicationCommandData().Options[0]

	eventId := data.Options[0].StringValue()
	attendance, err := attdnc.Get(eventId)
	if err != nil {
		if errors.Is(err, attdnc.ErrAttendanceNotFound) {
			return customerrors.InvalidAttendanceRecord
		}

		return errors.Wrap(err, "getting attendance record")
	}

	discordMembersList := data.Options[1:]

	for _, discordMember := range discordMembersList {
		if discordMember.UserValue(s).Bot {
			continue
		}

		member, err := members.Get(discordMember.UserValue(s).ID)
		if err != nil {
			if errors.Is(err, members.MemberNotFound) {
				_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: fmt.Sprintf("Member %s is not registered in the database and was not added to the attendance record.", discordMember.UserValue(s).Username),
					Flags:   discordgo.MessageFlagsEphemeral,
				})
				if err != nil {
					return errors.Wrap(err, "responding to interaction for unregistered member")
				}
				continue
			}

			return errors.Wrap(err, "getting member for add attendance")
		}

		if err := attendance.AddMember(member); err != nil {
			return errors.Wrap(err, "adding member to attendance")
		}
	}

	message, err := attendance.ToDiscordMessage()
	if err != nil {
		return errors.Wrap(err, "creating attendance message")
	}

	if _, err := s.ChannelMessageEditEmbeds(attendance.ChannelId, attendance.MessageId, message.Embeds); err != nil {
		return errors.Wrap(err, "sending attendance message")
	}

	if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: fmt.Sprintf("Attendance record https://discord.com/channels/%s/%s/%s updated", i.GuildID, settings.GetString("FEATURES.ATTENDANCE.CHANNEL_ID"), attendance.MessageId),
		Flags:   discordgo.MessageFlagsEphemeral,
	}); err != nil {
		return errors.Wrap(err, "sending followup message")
	}

	return nil
}

func addRemoveMembersAutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("taking attendance autocomplete")

	data := i.ApplicationCommandData()

	choices := []*discordgo.ApplicationCommandOptionChoice{}

	if data.Options[0].Options[0].Focused {
		attendanceRecords, err := attdnc.ListActive(5)
		if err != nil {
			return errors.Wrap(err, "getting active attendance records")
		}

		for _, record := range attendanceRecords {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  fmt.Sprintf("%s (%s)", record.Name, record.DateCreated.Local().Format("2006-01-02")),
				Value: record.Id,
			})
		}
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	}); err != nil {
		return errors.Wrap(err, "responding to take attendance auto complete")
	}

	return nil
}
