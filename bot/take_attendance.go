package bot

import (
	"context"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/rs/xid"
	attdnc "github.com/sol-armada/admin/attendance"
	"github.com/sol-armada/admin/members"
	"github.com/sol-armada/admin/settings"
	"github.com/sol-armada/admin/utils"
)

func takeAttendanceAutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("taking attendance autocomplete")

	data := i.ApplicationCommandData()

	choices := []*discordgo.ApplicationCommandOptionChoice{}

	switch {
	case !allowed(i.Member):
	case data.Options[0].Focused:
		attendanceRecords, err := attdnc.ListActive(5)
		if err != nil {
			return errors.Wrap(err, "getting active attendance records")
		}

		if data.Options[0].StringValue() != "" {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  data.Options[0].StringValue(),
				Value: data.Options[0].StringValue(),
			})
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

func takeAttendanceCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("taking attendance command")

	if !allowed(i.Member) {
		return InvalidPermissions
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	data := i.ApplicationCommandData()

	eventName := data.Options[0].StringValue()

	var attendance *attdnc.Attendance

	exists := false
	if _, err := xid.FromString(eventName); err == nil {
		exists = true
	}

	var err error
	if exists { // get an existing attendance record
		attendance, err = attdnc.Get(eventName)
	} else { // create a new attendance record
		attendance, err = attdnc.New(eventName)
	}
	if err != nil {
		return errors.Wrap(err, "getting or creating attendance record")
	}

	discordMembersList := data.Options[1:]

	for _, discordMember := range discordMembersList {
		member, err := members.Get(discordMember.UserValue(s).ID)
		if err != nil {
			if !errors.Is(err, members.MemberNotFound) {
				return errors.Wrap(err, "getting member for new attendance")
			}

			attendance.Issues = append(attendance.Issues, &attdnc.AttendanceIssue{
				Member: &members.Member{Id: discordMember.UserValue(s).ID},
				Reason: "Member not found in system",
			})

			continue
		}

		attendance.AddMember(member)
	}

	// check if the attendance record channel exists
	var channel *discordgo.Channel
	var message *discordgo.Message

	if attendance.ChannelId != "" {
		channel, _ = s.Channel(attendance.ChannelId)
	}

	if channel == nil { // if the channel doesn't exist, use the configured one instead
		channel, err = s.Channel(settings.GetString("FEATURES.ATTENDANCE.CHANNEL_ID"))
		if err != nil {
			return ChannelNotExist
		}
	}

	if attendance.MessageId != "" {
		message, _ = s.ChannelMessage(channel.ID, attendance.MessageId)
	}

	if message == nil {
		message, err = s.ChannelMessageSendComplex(channel.ID, attendance.ToDiscordMessage())
		if err != nil {
			return errors.Wrap(err, "sending attendance message")
		}
	}

	attendance.ChannelId = channel.ID
	attendance.MessageId = message.ID

	if err := attendance.Save(); err != nil {
		return errors.Wrap(err, "saving attendance record")
	}

	_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "Attendance record created!",
		Flags:   discordgo.MessageFlagsEphemeral,
	})

	return nil
}

func removeAttendanceAutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("removing attendance autocomplete")

	data := i.ApplicationCommandData()

	choices := []*discordgo.ApplicationCommandOptionChoice{}

	switch {
	case !allowed(i.Member):
	case data.Options[0].Focused:
		attendanceRecords, err := attdnc.ListActive(5)
		if err != nil {
			return errors.Wrap(err, "getting active attendance records")
		}

		if data.Options[0].StringValue() != "" {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  data.Options[0].StringValue(),
				Value: data.Options[0].StringValue(),
			})
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
		return errors.Wrap(err, "responding to remove attendance auto complete")
	}

	return nil
}

func removeAttendanceCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("removing attendance command")

	if !allowed(i.Member) {
		return InvalidPermissions
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	data := i.ApplicationCommandData()

	attendance, err := attdnc.Get(data.Options[0].StringValue())
	if err != nil {
		return errors.Wrap(err, "getting attendance record")
	}

	discordMembers := data.Options[1:]

	for _, discordMember := range discordMembers {
		member, err := members.Get(discordMember.UserValue(s).ID)
		if err != nil {
			if !errors.Is(err, members.MemberNotFound) {
				return errors.Wrap(err, "getting member for new attendance")
			}
			continue
		}
		attendance.RemoveMember(member)
	}

	if err := attendance.Save(); err != nil {
		return errors.Wrap(err, "saving attendance record")
	}

	message := attendance.ToDiscordMessage()

	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: attendance.ChannelId,
		ID:      attendance.MessageId,
		Content: &message.Content,
		Embeds:  &message.Embeds,
	}); err != nil {
		return errors.Wrap(err, "editing attendance message for member removal")
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

	if !allowed(i.Member) {
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

	if !allowed(i.Member) {
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

	return nil
}

func allowed(discordMember *discordgo.Member) bool {
	return utils.StringSliceContainsOneOf(discordMember.Roles, settings.GetStringSlice("FEATURES.ATTENDANCE.ALLOWED_ROLES"))
}
