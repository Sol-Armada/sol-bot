package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/rs/xid"
	attdnc "github.com/sol-armada/admin/attendance"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/members"
	"github.com/sol-armada/admin/ranks"
	"github.com/sol-armada/admin/utils"
)

func takeAttendanceAutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("taking attendance autocomplete")

	data := i.ApplicationCommandData()

	choices := []*discordgo.ApplicationCommandOptionChoice{}
	switch {
	case data.Options[0].Focused:
		channelMessages, err := s.ChannelMessages(config.GetString("FEATURES.ATTENDANCE.CHANNEL_ID"), 5, "", "", "")
		if err != nil {
			return errors.Wrap(err, "getting latest attendance messages for autocomplete")
		}
		if data.Options[0].StringValue() != "" {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  data.Options[0].StringValue(),
				Value: data.Options[0].StringValue(),
			})
		}
		for _, message := range channelMessages {
			if len(message.Reactions) > 0 {
				continue
			}

			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  message.Content,
				Value: message.ID,
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

	if !utils.StringSliceContainsOneOf(i.Member.Roles, config.GetStringSlice("ATTENDANCE.ALLOWED_ROLES")) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "You do not have permission to use this command",
			},
		}); err != nil {
			return errors.Wrap(err, "responding to take attendance command: invalid permissions")
		}

		return nil
	}

	data := i.ApplicationCommandData()

	eventName := strings.TrimPrefix(data.Options[0].StringValue(), "(NEW) ")
	memberIds := data.Options[1:]

	attendance := &attdnc.Attendance{
		Id:     xid.New().String(),
		Name:   eventName,
		Issues: []*attdnc.AttendanceIssue{},
	}

	channelId := config.GetString("FEATURES.ATTENDANCE.CHANNEL_ID")

	membersList := []*members.Member{}
	for _, memberId := range memberIds {
		member, err := members.Get(memberId.UserValue(s).ID)
		if err != nil {
			// this should never happen
			attendance.Issues = append(attendance.Issues, &attdnc.AttendanceIssue{
				Member: &members.Member{
					Id:   memberId.UserValue(s).ID,
					Name: memberId.UserValue(s).Username,
				},
				Reason: "not in system",
			})
			continue
		}

		if len(attdnc.Issues(member)) > 0 {
			attendance.Issues = append(attendance.Issues, &attdnc.AttendanceIssue{
				Member: member,
				Reason: strings.Join(attdnc.Issues(member), ", "),
			})
			continue
		}

		membersList = append(membersList, member)
	}

	attendance.Members = membersList

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "Attendance taken",
		},
	}); err != nil {
		// TODO: just log this
		return errors.Wrap(err, "responding to take attendance command")
	}

	var eventMessage *discordgo.Message
	eventMessage, _ = s.ChannelMessage(channelId, attendance.Name)

	if eventMessage == nil {
		// create a new attendance message
		eventMessage, err := s.ChannelMessageSend(channelId, fmt.Sprintf("%s (%s)", attendance.Name, xid.New().String()))
		if err != nil {
			return errors.Wrap(err, "creating new attendance message")
		}

		emThread, err := s.MessageThreadStart(eventMessage.ChannelID, eventMessage.ID, attendance.Name+" Attendance Thread", 1440)
		if err != nil {
			return errors.Wrap(err, "creating new attendance thread")
		}

		eventMessage.Thread = emThread
	}

	eventThreadMessages, err := s.ChannelMessages(eventMessage.Thread.ID, 100, "", "", "")
	if err != nil {
		return errors.Wrap(err, "getting attendance thread messages")
	}
	log.WithField("messages", eventThreadMessages).Debug("event message list")

	// we need to create a new message
	if len(eventThreadMessages) == 1 {
		// make the primary list
		_, err := s.ChannelMessageSendComplex(eventMessage.Thread.ID, &discordgo.MessageSend{
			Content: attendance.GenerateList(),
			Embeds:  []*discordgo.MessageEmbed{attendance.GetIssuesEmbed()},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Record Attendance",
							Style:    discordgo.PrimaryButton,
							CustomID: "attendance:record:" + attendance.Id,
							Emoji:    &discordgo.ComponentEmoji{Name: "📝"},
						},
						discordgo.Button{
							Label:    "Recheck Issues",
							Style:    discordgo.SecondaryButton,
							CustomID: "attendance:recheck:" + attendance.Id,
							Emoji:    &discordgo.ComponentEmoji{Name: "🔄"},
						},
					},
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "creating attendance thread message")
		}

		return nil
	}

	// we have a message already
	message := eventThreadMessages[len(eventThreadMessages)-2]
	currentUsersSplit := strings.Split(message.Content, "\n")
	currentUsersSplit = append(currentUsersSplit, strings.Split(message.Embeds[0].Fields[0].Value, "\n")...)
	for _, cu := range currentUsersSplit[1:] {
		if cu == "No members" || cu == "" {
			continue
		}
		uid := strings.ReplaceAll(cu, "<@", "")
		uid = strings.ReplaceAll(uid, ">", "")
		uid = strings.Split(uid, ":")[0]

		member, err := members.Get(uid)
		if err != nil {
			return errors.Wrap(err, "getting member from existing attendance")
		}
		attendance.AddMember(member)
	}

	attendaceList := attendance.GenerateList()

	emb := []*discordgo.MessageEmbed{
		attendance.GetIssuesEmbed(),
	}
	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: eventMessage.Thread.ID,
		ID:      message.ID,
		Content: &attendaceList,
		Embeds:  &emb,
	}); err != nil {
		return errors.Wrap(err, "editing attendance thread message")
	}

	return nil
}

func removeAttendanceAutocompleteHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("removing attendance autocomplete")

	data := i.ApplicationCommandData()

	choices := []*discordgo.ApplicationCommandOptionChoice{}
	switch {
	case data.Options[0].Focused:
		channelMessages, err := s.ChannelMessages(config.GetString("FEATURES.ATTENDANCE.CHANNEL_ID"), 5, "", "", "")
		if err != nil {
			return errors.Wrap(err, "getting latest messages for remove auto complete")
		}
		for _, message := range channelMessages {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  message.Content,
				Value: message.ID,
			})
		}
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	}); err != nil {
		return errors.Wrap(err, "responding to removeattendance autocomplete")
	}

	return nil
}

func removeAttendanceCommandHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("removing attendance command")

	if !utils.StringSliceContainsOneOf(i.Member.Roles, config.GetStringSlice("ATTENDANCE.ALLOWED_ROLES")) {
		logger.Debug("invalid permissions")

		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "You do not have permission to use this command",
			},
		}); err != nil {
			return errors.Wrap(err, "responding to onboarding command invalid permissions")
		}

		return nil
	}

	data := i.ApplicationCommandData()

	eventName := data.Options[0]
	userIds := data.Options[1:]

	channelId := config.GetString("FEATURES.ATTENDANCE.CHANNEL_ID")
	em, err := s.ChannelMessage(channelId, eventName.StringValue())
	if err != nil {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Event not found",
			},
		}); err != nil {
			return errors.Wrap(err, "responding with event not found")
		}

		return nil
	}

	etms, err := s.ChannelMessages(em.Thread.ID, 10, "", "", "")
	if err != nil {
		return errors.Wrap(err, "getting attendance thread messages")
	}

	// remove all members in usersList from currentUsersSplit
	message := etms[len(etms)-2]
	currentUsersSplit := strings.Split(message.Content, "\n")
	currentUsersSplit = append(currentUsersSplit, strings.Split(message.Embeds[0].Fields[0].Value, "\n")...)

	removedIds := ""
	for _, user := range userIds {
		removedIds += user.UserValue(s).ID + ","
	}

	logger.WithField("ids", removedIds).Debug("removing from attendance")

	attendance := &attdnc.Attendance{}
	for _, discordUserId := range currentUsersSplit[1:] {
		if discordUserId == "No members" {
			continue
		}

		discordUserId = strings.ReplaceAll(discordUserId, "<@", "")
		discordUserId = strings.ReplaceAll(discordUserId, ">", "")
		discordUserId = strings.Split(discordUserId, ":")[0]

		if strings.Contains(removedIds, discordUserId) {
			continue
		}

		member, err := members.Get(discordUserId)
		if err != nil {
			return errors.Wrap(err, "getting user for removing from attendance")
		}

		if len(attdnc.Issues(member)) > 0 {
			attendance.Issues = append(attendance.Issues, &attdnc.AttendanceIssue{
				Member: member,
				Reason: strings.Join(attdnc.Issues(member), ", "),
			})
			continue
		}

		attendance.AddMember(member)
	}

	attendaceList := attendance.GenerateList()

	emb := []*discordgo.MessageEmbed{
		attendance.GetIssuesEmbed(),
	}
	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: em.Thread.ID,
		ID:      message.ID,
		Content: &attendaceList,
		Embeds:  &emb,
	}); err != nil {
		return errors.Wrap(err, "editing attendance thread message")
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "Removed from attendance",
		},
	}); err != nil {
		return errors.Wrap(err, "responding to takeattendance command")
	}

	return nil
}

func recordAttendanceButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("recording attendance button handler")

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	threadId := i.Message.ChannelID

	threadMessages, err := s.ChannelMessages(threadId, 100, "", "", "")
	if err != nil {
		return errors.Wrap(err, "getting attendance thread messages")
	}

	attendance := &attdnc.Attendance{}
	attendance.NewFromThreadMessages(threadMessages)

	logger.WithField("members", attendance.Members).Debug("marking for attendance")

	rankUps := ""
	for _, u := range attendance.Members {
		u.IncrementEventCount()

		switch u.Rank {
		case ranks.Recruit:
			if u.Events >= 3 {
				rankUps += fmt.Sprintf("<@%s> has made Member\n", u.Id)
			}
		case ranks.Member:
			if u.Events >= 10 {
				rankUps += fmt.Sprintf("<@%s> has made Technician\n", u.Id)
			}
		case ranks.Technician:
			if u.Events >= 20 {
				rankUps += fmt.Sprintf("<@%s> has made Specialist\n", u.Id)
			}
		}
	}
	if rankUps != "" {
		rankUps += "\nDon't forget to rank these members!"

		_, _ = s.ChannelMessageSend(threadId, rankUps)
	}

	comp := []discordgo.MessageComponent{}
	_, _ = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    threadId,
		ID:         threadMessages[len(threadMessages)-2].ID,
		Components: &comp,
	})

	parentMessage := threadMessages[len(threadMessages)-1].MessageReference
	_ = s.MessageReactionAdd(parentMessage.ChannelID, parentMessage.MessageID, "✅")

	return nil
}

func recheckIssuesButtonHandler(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger := utils.GetLoggerFromContext(ctx).(*log.Entry)
	logger.Debug("rechecking issues button handler")

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	threadId := i.Message.ChannelID

	threadMessages, err := s.ChannelMessages(threadId, 100, "", "", "")
	if err != nil {
		return errors.Wrap(err, "getting attendance thread messages")
	}

	attendance := &attdnc.Attendance{}
	attendance.NewFromThreadMessages(threadMessages)

	membersList := []*members.Member{}
	for _, aMember := range attendance.Members {
		member, err := members.Get(aMember.Id)
		if err != nil {
			attendance.Issues = append(attendance.Issues, &attdnc.AttendanceIssue{
				Member: &members.Member{
					Id:   aMember.Id,
					Name: aMember.Name,
				},
				Reason: "not in system",
			})
			continue
		}

		if len(attdnc.Issues(member)) > 0 {
			attendance.Issues = append(attendance.Issues, &attdnc.AttendanceIssue{
				Member: member,
				Reason: strings.Join(attdnc.Issues(member), ", "),
			})
			continue
		}

		membersList = append(membersList, member)
	}

	attendance.Members = membersList

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "Attendance taken",
		},
	}); err != nil {
		logger.WithError(err).Error("responding to recheck issues button")
	}

	etms, err := s.ChannelMessages(i.ChannelID, 100, "", "", "")
	if err != nil {
		return errors.Wrap(err, "getting attendance thread messages")
	}
	log.WithField("messages", etms).Debug("event message list")

	attendaceList := attendance.GenerateList()

	emb := []*discordgo.MessageEmbed{
		attendance.GetIssuesEmbed(),
	}
	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: i.ChannelID,
		ID:      etms[len(etms)-2].ID,
		Content: &attendaceList,
		Embeds:  &emb,
	}); err != nil {
		return errors.Wrap(err, "editing attendance thread message")
	}

	return nil
}
