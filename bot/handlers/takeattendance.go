package handlers

import (
	"fmt"
	"slices"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/xid"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/ranks"
	"github.com/sol-armada/admin/users"
)

func TakeAttendanceAutocompleteHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()

	choices := []*discordgo.ApplicationCommandOptionChoice{}
	switch {
	case data.Options[0].Focused:
		channelMessages, err := s.ChannelMessages(config.GetString("DISCORD.CHANNELS.ATTENDANCE"), 5, "", "", "")
		if err != nil {
			log.WithError(err).Error("getting all attendance messages")
			return
		}
		if data.Options[0].StringValue() != "" {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  "(NEW) " + data.Options[0].StringValue(),
				Value: data.Options[0].StringValue(),
			})
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
		log.WithError(err).Error("responding to takeattendance command")
		return
	}
}

func TakeAttendanceCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user, err := users.Get(i.Member.User.ID)
	if err != nil {
		log.WithError(err).Error("getting user")
		return
	}

	if user.Rank > ranks.GetRankByName(config.GetStringWithDefault("FEATURES.ATTENDANCE.MIN_RANK", "admiral")) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "You do not have permission to use this command",
			},
		}); err != nil {
			log.WithError(err).Error("responding to onboarding command")
			return
		}
		return
	}

	data := i.ApplicationCommandData()

	eventName := data.Options[0]
	userIds := data.Options[1:]

	channelId := config.GetString("DISCORD.CHANNELS.ATTENDANCE")

	usersList := []string{}
	for _, userId := range userIds {
		usersList = append(usersList, fmt.Sprintf("<@%s>", userId.UserValue(s).ID))
	}

	var em *discordgo.Message
	em, _ = s.ChannelMessage(channelId, eventName.StringValue())

	if em == nil {
		// create a new attendance message
		em, err = s.ChannelMessageSend(channelId, fmt.Sprintf("%s (%s)", eventName.StringValue(), xid.New().String()))
		if err != nil {
			log.WithError(err).Error("creating new attendance message")
			return
		}

		emThread, err := s.MessageThreadStart(em.ChannelID, em.ID, eventName.StringValue()+" Attendance Thread", 1440)
		if err != nil {
			log.WithError(err).Error("creating new attendance message")
			return
		}

		em.Thread = emThread
	}

	etms, err := s.ChannelMessages(em.Thread.ID, 10, "", "", "")
	if err != nil {
		log.WithError(err).Error("getting attendance thread messages")
		return
	}

	if len(etms) == 1 {
		slices.Sort(usersList)

		_, err := s.ChannelMessageSend(em.Thread.ID, strings.Join(usersList, "\n"))
		if err != nil {
			log.WithError(err).Error("creating attendance thread message")
			return
		}
	} else {
		currentUsersSplit := strings.Split(etms[0].Content, "\n")
		usersList = append(currentUsersSplit, usersList...)
		usersList = removeDuplicates(usersList)

		slices.Sort(usersList)

		if _, err := s.ChannelMessageEdit(em.Thread.ID, etms[0].ID, strings.Join(usersList, "\n")); err != nil {
			log.WithError(err).Error("editing attendance thread message")
			return
		}
	}

	takeAttendanceComplete(s, i)
}

func RemoveAttendanceAutocompleteHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()

	choices := []*discordgo.ApplicationCommandOptionChoice{}
	switch {
	case data.Options[0].Focused:
		channelMessages, err := s.ChannelMessages(config.GetString("DISCORD.CHANNELS.ATTENDANCE"), 5, "", "", "")
		if err != nil {
			log.WithError(err).Error("getting all attendance messages")
			return
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
		log.WithError(err).Error("responding to removeattendance command")
		return
	}
}

func RemoveAttendanceCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user, err := users.Get(i.Member.User.ID)
	if err != nil {
		log.WithError(err).Error("getting user")
		return
	}

	if user.Rank > ranks.GetRankByName(config.GetStringWithDefault("FEATURES.ATTENDANCE.MIN_RANK", "admiral")) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "You do not have permission to use this command",
			},
		}); err != nil {
			log.WithError(err).Error("responding to onboarding command")
			return
		}
		return
	}

	data := i.ApplicationCommandData()

	eventName := data.Options[0]
	userIds := data.Options[1:]

	channelId := config.GetString("DISCORD.CHANNELS.ATTENDANCE")

	usersList := []string{}
	for _, userId := range userIds {
		usersList = append(usersList, fmt.Sprintf("<@%s>", userId.UserValue(s).ID))
	}

	em, err := s.ChannelMessage(channelId, eventName.StringValue())
	if err != nil {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Event not found",
			},
		}); err != nil {
			log.WithError(err).Error("responding to onboarding command")
		}
		return
	}

	etms, err := s.ChannelMessages(em.Thread.ID, 10, "", "", "")
	if err != nil {
		log.WithError(err).Error("getting attendance thread messages")
		return
	}

	// remove all users from usersList from currentUsersSplit
	currentUsersSplit := strings.Split(etms[0].Content, "\n")
	newUsersSlice := []string{}
	for _, user := range currentUsersSplit {
		if !strings.Contains(strings.Join(usersList, "\n"), user) {
			newUsersSlice = append(newUsersSlice, user)
		}
	}

	slices.Sort(newUsersSlice)

	if _, err := s.ChannelMessageEdit(em.Thread.ID, etms[0].ID, strings.Join(newUsersSlice, "\n")); err != nil {
		log.WithError(err).Error("editing attendance thread message")
		return
	}

	removeAttendanceComplete(s, i)
}

func removeDuplicates(s []string) []string {
	m := make(map[string]bool)
	for _, item := range s {
		m[item] = true
	}

	items := make([]string, 0, len(m))
	for item := range m {
		items = append(items, item)
	}

	return items
}

func takeAttendanceComplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "Attendance taken",
		},
	}); err != nil {
		log.WithError(err).Error("responding to takeattendance command")
		return
	}
}

func removeAttendanceComplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "Removed from attendance",
		},
	}); err != nil {
		log.WithError(err).Error("responding to takeattendance command")
		return
	}
}
