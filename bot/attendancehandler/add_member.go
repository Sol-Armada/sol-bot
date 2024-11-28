package attendancehandler

import (
	"context"
	"fmt"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	attdnc "github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/customerrors"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/utils"
)

func AddMembersCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
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
			if !errors.Is(err, members.MemberNotFound) {
				return errors.Wrap(err, "getting member for new attendance")
			}

			attendance.WithIssues = append(attendance.WithIssues, member)

			continue
		}

		attendance.AddMember(member)
	}

	// save now incase there is an error with creating the message
	if err := attendance.Save(); err != nil {
		return errors.Wrap(err, "saving attendance record")
	}

	attandanceMessage := attendance.ToDiscordMessage()
	if _, err := s.ChannelMessageEditEmbeds(attendance.ChannelId, attendance.MessageId, attandanceMessage.Embeds); err != nil {
		return errors.Wrap(err, "sending attendance message")
	}

	_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: fmt.Sprintf("Attendance record https://discord.com/channels/%s/%s/%s updated", i.GuildID, settings.GetString("FEATURES.ATTENDANCE.CHANNEL_ID"), attendance.MessageId),
		Flags:   discordgo.MessageFlagsEphemeral,
	})

	return nil
}

func AddRemoveMembersAutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("taking attendance autocomplete")

	data := i.ApplicationCommandData()

	choices := []*discordgo.ApplicationCommandOptionChoice{}

	switch {
	case !allowed(i.Member, "ATTENDANCE"):
	case data.Options[0].Options[0].Focused:
		attendanceRecords, err := attdnc.ListActive(5)
		if err != nil {
			return errors.Wrap(err, "getting active attendance records")
		}

		for _, record := range attendanceRecords {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  record.Name,
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
