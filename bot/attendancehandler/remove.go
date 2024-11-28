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

func RemoveMembersCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("remove member attendance command")

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
				return errors.Wrap(err, "getting member for removing attendance")
			}

			attendance.WithIssues = append(attendance.WithIssues, member)

			continue
		}

		attendance.RemoveMember(member)
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
