package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/rs/xid"
	attdnc "github.com/sol-armada/sol-bot/attendance"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/utils"
	"go.mongodb.org/mongo-driver/bson"
)

var attendanceSubCommands = map[string]Handler{
	"new":     newAttendanceCommandHandler,
	"add":     addMembersAttendanceCommandHandler,
	"remove":  removeMembersAttendanceCommandHandler,
	"refresh": refreshAttendanceCommandHandler,
	"revert":  revertAttendanceCommandHandler,
}

var lastRefreshTime time.Time

func attendanceCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("attendance command handler")

	if !allowed(i.Member, "ATTENDANCE") {
		return InvalidPermissions
	}

	data := i.Interaction.ApplicationCommandData()
	handler, ok := attendanceSubCommands[data.Options[0].Name]
	if !ok {
		return InvalidSubcommand
	}

	return handler(ctx, s, i)
}

func newAttendanceCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("new attendance command")

	commandMember := utils.GetMemberFromContext(ctx).(*members.Member)

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	data := i.Interaction.ApplicationCommandData().Options[0]

	eventName := data.Options[0].StringValue()

	exists := false
	if _, err := xid.FromString(eventName); err == nil {
		exists = true
	}

	attendance := attdnc.New(eventName, commandMember)

	discordMembersList := data.Options[1:]

	for _, discordMember := range discordMembersList {
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
	message, err := s.ChannelMessageSendComplex(attendance.ChannelId, attandanceMessage)
	if err != nil {
		return errors.Wrap(err, "sending attendance message")
	}
	attendance.MessageId = message.ID

	if err := attendance.Save(); err != nil {
		return errors.Wrap(err, "saving attendance record")
	}

	content := "Attendance record created!"
	if exists {
		content = "Attendance record updated!"
	}
	_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: content,
		Flags:   discordgo.MessageFlagsEphemeral,
	})

	return nil
}

func addMembersAttendanceCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
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
			return InvalidAttendanceRecord
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
		Content: "Attendance record updated!",
		Flags:   discordgo.MessageFlagsEphemeral,
	})

	return nil
}

func addRemoveMembersAttendanceAutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
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

func removeMembersAttendanceCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
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
			return InvalidAttendanceRecord
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
		Content: "Attendance record updated!",
		Flags:   discordgo.MessageFlagsEphemeral,
	})

	return nil
}

func recheckIssuesButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("rechecking issues button handler")

	if !allowed(i.Member, "ATTENDANCE") {
		return InvalidPermissions
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	id := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	attendance, err := attdnc.Get(id)
	if err != nil {
		return errors.Wrap(err, "getting attendance record")
	}

	if err := attendance.RecheckIssues(); err != nil {
		return errors.Wrap(err, "rechecking issues for attendance record")
	}

	message := attendance.ToDiscordMessage()

	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: attendance.ChannelId,
		ID:      attendance.MessageId,
		Content: &message.Content,
		Embeds:  &message.Embeds,
	}); err != nil {
		return errors.Wrap(err, "editing attendance message for rechecking issues")
	}

	_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "Attendance record updated!",
		Flags:   discordgo.MessageFlagsEphemeral,
	})

	return nil
}

func recordAttendanceButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("recording attendance button handler")

	if !allowed(i.Member, "ATTENDANCE") {
		return InvalidPermissions
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	id := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	attendance, err := attdnc.Get(id)
	if err != nil {
		return errors.Wrap(err, "getting attendance record")
	}

	if err := attendance.Record(); err != nil {
		return errors.Wrap(err, "recording attendance for attendance record")
	}

	attendanceMessage := attendance.ToDiscordMessage()
	_, _ = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    attendance.ChannelId,
		ID:         attendance.MessageId,
		Content:    &attendanceMessage.Content,
		Embeds:     &attendanceMessage.Embeds,
		Components: &[]discordgo.MessageComponent{},
	})

	return nil
}

func deleteAttendanceButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("deleting attendance button handler")

	if !allowed(i.Member, "ATTENDANCE") {
		return InvalidPermissions
	}

	id := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	attendance, err := attdnc.Get(id)
	if err != nil && !errors.Is(err, attdnc.ErrAttendanceNotFound) {
		return errors.Wrap(err, "getting attendance record")
	}
	if attendance == nil {
		_ = s.ChannelMessageDelete(i.ChannelID, i.Message.ID)

		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Looks like that attendance doesn't exist in the database anyway, removed the message.",
			},
		})
		return nil
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "Are you sure you want to delete this attendance record?",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Yes",
							Style:    discordgo.DangerButton,
							CustomID: fmt.Sprintf("attendance:verifydelete:%s", id),
						},
						discordgo.Button{
							Label:    "No",
							Style:    discordgo.SecondaryButton,
							CustomID: fmt.Sprintf("attendance:canceldelete:%s", id),
						},
					},
				},
			},
		},
	})

	return nil
}

func verifyDeleteButtonModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("deleting verify modal handler")

	if !allowed(i.Member, "ATTENDANCE") {
		return InvalidPermissions
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	id := strings.Split(i.MessageComponentData().CustomID, ":")[2]

	attendance, err := attdnc.Get(id)
	if err != nil && !errors.Is(err, attdnc.ErrAttendanceNotFound) {
		return errors.Wrap(err, "getting attendance record")
	}
	if attendance != nil {
		if err := attendance.Delete(); err != nil {
			return errors.Wrap(err, "deleting attendance record")
		}

		if err := s.ChannelMessageDelete(attendance.ChannelId, attendance.MessageId); err != nil {
			derr := err.(*discordgo.RESTError)
			if derr.Response.StatusCode != 404 {
				return errors.Wrap(err, "deleting attendance message")
			}
		}
	}

	_, _ = s.FollowupMessageEdit(i.Interaction, i.Message.ID, &discordgo.WebhookEdit{
		Content:    utils.StringPointer("Attendance record deleted!"),
		Components: &[]discordgo.MessageComponent{},
	})

	return nil
}

func cancelDeleteButtonModalHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("deleting cancel modal handler")

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "Whew, that was close. Attendance record not deleted.",
		},
	})

	return nil
}

func refreshAttendanceCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("refreshing attendance command handler")

	if !allowed(i.Member, "ATTENDANCE") {
		return InvalidPermissions
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	if lastRefreshTime.After(time.Now().Add(-1 * time.Hour)) {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Already refreshed in the last hour!",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return nil
	}

	attendance, err := attdnc.List(bson.D{}, 10, 1)
	if err != nil {
		return errors.Wrap(err, "getting attendance records")
	}

	m, _ := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "Attendance refreshing...",
		Flags:   discordgo.MessageFlagsEphemeral,
	})

	for idx, a := range attendance {
		msg := a.ToDiscordMessage()
		if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Channel:    a.ChannelId,
			ID:         a.MessageId,
			Embeds:     &msg.Embeds,
			Components: &msg.Components,
		}); err != nil {
			return err
		}

		_, _ = s.FollowupMessageEdit(i.Interaction, m.ID, &discordgo.WebhookEdit{
			Content: utils.StringPointer(fmt.Sprintf("Attendance refreshing... (%d/%d)", idx+1, len(attendance))),
		})

		time.Sleep(250 * time.Millisecond)
	}

	_, _ = s.FollowupMessageEdit(i.Interaction, m.ID, &discordgo.WebhookEdit{
		Content: utils.StringPointer("Attendance refreshed!"),
	})

	lastRefreshTime = time.Now()

	return nil
}

func revertAttendanceAutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("taking attendance autocomplete")

	data := i.ApplicationCommandData()

	choices := []*discordgo.ApplicationCommandOptionChoice{}

	switch {
	case !allowed(i.Member, "ATTENDANCE"):
	case data.Options[0].Options[0].Focused:
		attendanceRecords, err := attdnc.List(bson.D{}, 10, 1)
		if err != nil {
			return errors.Wrap(err, "getting recorded attendance records")
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
		return errors.Wrap(err, "responding to revert attendance auto complete")
	}

	return nil
}

func revertAttendanceCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("reverting attendance command handler")

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	data := i.Interaction.ApplicationCommandData()
	id := data.Options[0].Options[0].StringValue()

	attendance, err := attdnc.Get(id)
	if err != nil {
		return errors.Wrap(err, "getting attendance record")
	}

	msg, _ := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "Reverting attendance...",
		Flags:   discordgo.MessageFlagsEphemeral,
	})

	if err := attendance.Revert(); err != nil {
		return errors.Wrap(err, "reverting attendance")
	}

	attendanceMessage, err := s.ChannelMessage(attendance.ChannelId, attendance.MessageId)
	if err != nil {
		if derr, ok := err.(*discordgo.RESTError); ok {
			if derr.Response.StatusCode == 404 {
				_, _ = s.FollowupMessageEdit(i.Interaction, msg.ID, &discordgo.WebhookEdit{
					Content: utils.StringPointer("It looks like that attendance record message is missing! Creating it again..."),
				})

				if _, err := s.ChannelMessageSendComplex(attendance.ChannelId, &discordgo.MessageSend{
					Content:    "Recreated because message was missing!",
					Embeds:     attendance.ToDiscordMessage().Embeds,
					Components: attendance.ToDiscordMessage().Components,
				}); err != nil {
					return err
				}

				return nil
			}
		}

		return errors.Wrap(err, "getting attendance message")
	}

	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    attendanceMessage.ChannelID,
		ID:         attendanceMessage.ID,
		Embeds:     &attendance.ToDiscordMessage().Embeds,
		Components: &attendance.ToDiscordMessage().Components,
	}); err != nil {
		return err
	}

	_, _ = s.FollowupMessageEdit(i.Interaction, msg.ID, &discordgo.WebhookEdit{
		Content: utils.StringPointer("Attendance reverted!"),
	})

	return nil
}

func allowed(discordMember *discordgo.Member, feature string) bool {
	return utils.StringSliceContainsOneOf(discordMember.Roles, settings.GetStringSlice("FEATURES."+feature+".ALLOWED_ROLES"))
}
