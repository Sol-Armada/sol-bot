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

func removeMembersCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx)
	logger.Debug("remove member attendance command")

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		return errors.Wrap(err, "responding to interaction")
	}

	data := i.Interaction.ApplicationCommandData().Options[0]

	eventId := data.Options[0].StringValue()
	a, err := attdnc.Get(eventId)
	if err != nil {
		if errors.Is(err, attdnc.ErrAttendanceNotFound) {
			return customerrors.InvalidAttendanceRecord
		}

		return errors.Wrap(err, "getting attendance record")
	}

	discordMembersList := data.Options[1:]

	for _, discordMember := range discordMembersList {
		member, err := members.Get(discordMember.StringValue())
		if err != nil {
			if errors.Is(err, members.MemberNotFound) {
				_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: fmt.Sprintf("Member %s is not registered in the database and was not removed from the attendance record.", discordMember.StringValue()),
					Flags:   discordgo.MessageFlagsEphemeral,
				})
				if err != nil {
					return errors.Wrap(err, "responding to interaction for unregistered member")
				}
				continue
			}

			return errors.Wrap(err, "getting member for remove attendance")
		}

		if err := a.RemoveParticipant(member); err != nil {
			return errors.Wrap(err, "removing participant from attendance record")
		}
	}

	message, err := a.ToDiscordMessage()
	if err != nil {
		return errors.Wrap(err, "creating attendance message")
	}

	if _, err := s.ChannelMessageEditEmbeds(a.ChannelId, a.MessageId, message.Embeds); err != nil {
		return errors.Wrap(err, "sending attendance message")
	}

	if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: fmt.Sprintf("Attendance record https://discord.com/channels/%s/%s/%s updated", i.GuildID, settings.GetString("FEATURES.ATTENDANCE.CHANNEL_ID"), a.MessageId),
		Flags:   discordgo.MessageFlagsEphemeral,
	}); err != nil {
		return errors.Wrap(err, "sending follow-up message")
	}

	return nil
}
