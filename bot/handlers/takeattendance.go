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
	userIssues := map[string][]string{}
	for _, userId := range userIds {
		user, err := users.Get(userId.UserValue(s).ID)
		if err != nil {
			log.WithError(err).Error("getting user for attendance")
			return
		}

		if len(user.Issues()) > 0 {
			userIssues[user.ID] = user.Issues()
			continue
		}

		usersList = append(usersList, fmt.Sprintf("<@%s>", userId.UserValue(s).ID))
		// user.IncrementEventCount()
	}
	takeAttendanceComplete(s, i, userIssues)

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

	etms, err := s.ChannelMessages(em.Thread.ID, 100, "", "", "")
	if err != nil {
		log.WithError(err).Error("getting attendance thread messages")
		return
	}
	log.WithField("messages", etms).Debug("event message list")

	if len(etms) == 1 {
		slices.Sort(usersList)
		m := strings.Join(usersList, "\n")

		// if len(userIssues) > 0 {
		// 	m += "\n\nWith Issues\n"
		// 	for k, v := range userIssues {
		// 		m += fmt.Sprintf("<@%s>: %s\n", k, strings.Join(v, ", "))
		// 	}
		// }

		_, err := s.ChannelMessageSend(em.Thread.ID, m)
		if err != nil {
			log.WithError(err).Error("creating attendance thread message")
			return
		}
	} else {
		message := etms[len(etms)-2]
		log.WithField("content", message.Content).Debug("editing attendance thread message")
		currentUsersSplit := strings.Split(message.Content, "\n")
		for _, cu := range currentUsersSplit {
			if cu == "" {
				continue
			}
			usersList = append(usersList, cu)
			if strings.Contains(cu, "With Issues") {
				break
			}
		}
		usersList = removeDuplicates(usersList)
		m := strings.Join(usersList, "\n")

		// if len(userIssues) > 0 {
		// 	m += "\n\nWith Issues\n"
		// 	for k, v := range userIssues {
		// 		m += fmt.Sprintf("<@%s>: %s\n", k, strings.Join(v, ", "))
		// 	}
		// }

		if _, err := s.ChannelMessageEdit(em.Thread.ID, message.ID, m); err != nil {
			log.WithError(err).Error("editing attendance thread message")
			return
		}
	}

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

	// remove all users in usersList from currentUsersSplit
	message := etms[len(etms)-2]
	currentUsersSplit := strings.Split(message.Content, "\n")
	newUsersSlice := []string{}
	for _, user := range currentUsersSplit {
		if !strings.Contains(strings.Join(usersList, "\n"), user) {
			newUsersSlice = append(newUsersSlice, user)
			continue
		}

		// userId := strings.ReplaceAll(user, "<@", "")
		// userId = strings.ReplaceAll(userId, ">", "")
		// user, err := users.Get(userId)
		// if err != nil {
		// 	log.WithError(err).Error("getting user")
		// 	return
		// }

		// user.DecrementEventCount()
	}

	if _, err := s.ChannelMessageEdit(em.Thread.ID, message.ID, strings.Join(newUsersSlice, "\n")); err != nil {
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

func takeAttendanceComplete(s *discordgo.Session, i *discordgo.InteractionCreate, userIssues map[string][]string) {
	responseData := &discordgo.InteractionResponseData{
		Flags:   discordgo.MessageFlagsEphemeral,
		Content: "Attendance taken",
	}
	if len(userIssues) > 0 {
		e := discordgo.MessageEmbed{}
		e.Title = "Attendance Issues"
		fieldValue := ""
		for k, v := range userIssues {
			fieldValue += fmt.Sprintf("<@%s>: %s\n", k, strings.Join(v, ", "))
		}
		e = discordgo.MessageEmbed{
			Title:       "Attendance Issues",
			Description: "Member issues preventing event credit",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Member - Issues",
					Value: fieldValue,
				},
			},
		}
		responseData.Embeds = []*discordgo.MessageEmbed{&e}
	}
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: responseData,
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
