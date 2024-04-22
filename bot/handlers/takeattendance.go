package handlers

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/xid"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/members"
	"github.com/sol-armada/admin/ranks"
)

type AttendanceIssue struct {
	Member *members.Member
	Reason string
}

type Attendance struct {
	Id      string
	Name    string
	Members []*members.Member
	Issues  []*AttendanceIssue
}

func (a *Attendance) GenerateList() string {
	// remove duplicates
	list := make(map[string]*members.Member)
	for _, u := range a.Members {
		list[u.Id] = u
	}

	a.Members = []*members.Member{}
	for _, u := range list {
		a.Members = append(a.Members, u)
	}

	slices.SortFunc(a.Members, func(a, b *members.Member) int {
		if a.Rank > b.Rank {
			return 1
		}
		if a.Rank < b.Rank {
			return -1
		}
		if a.Name < b.Name {
			return 1
		}
		if a.Name > b.Name {
			return -1
		}

		return 0
	})

	m := ""
	for i, u := range a.Members {
		m += fmt.Sprintf("<@%s>", u.Id)
		if i < len(a.Members)-1 {
			m += "\n"
		}
	}

	if m == "" {
		m = "No members"
	}

	return "Attendance List:\n" + m
}

func (a *Attendance) removeDuplicates() {
	list := []*members.Member{}

	for _, u := range a.Members {
		found := false
		for _, v := range list {
			if u.Id == v.Id {
				found = true
				break
			}
		}
		if found {
			continue
		}
		list = append(list, u)
	}

	a.Members = list
}

func (a *Attendance) getIssuesEmbed() *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       "Users with Issues",
		Description: "List of members with attendance credit issues",
		Fields:      []*discordgo.MessageEmbedField{},
	}

	fieldValue := ""
	for _, issue := range a.Issues {
		fieldValue += fmt.Sprintf("<@%s>: %s\n", issue.Member.Id, issue.Reason)
	}
	field := &discordgo.MessageEmbedField{
		Name:  "Member - Issues",
		Value: fieldValue,
	}
	embed.Fields = append(embed.Fields, field)

	return embed
}

func (a *Attendance) Parse(threadMessages []*discordgo.Message) {
	mainMessage := threadMessages[len(threadMessages)-1].ReferencedMessage
	attendanceMessage := threadMessages[len(threadMessages)-2]

	// get the ID between ( )
	reg := regexp.MustCompile(`(.*?)\((.*?)\)`)
	a.Id = reg.FindStringSubmatch(mainMessage.Content)[1]

	// get the name before ( )
	a.Name = reg.FindStringSubmatch(mainMessage.Content)[0]

	currentUsersSplit := strings.Split(attendanceMessage.Content, "\n")
	currentUsersSplit = append(currentUsersSplit, strings.Split(attendanceMessage.Embeds[0].Fields[0].Value, "\n")...)
	for _, cu := range currentUsersSplit[1:] {
		if cu == "No members" || cu == "" {
			continue
		}
		uid := strings.ReplaceAll(cu, "<@", "")
		uid = strings.ReplaceAll(uid, ">", "")
		uid = strings.Split(uid, ":")[0]

		u, err := members.Get(uid)
		if err != nil {
			log.WithError(err).Error("getting user for existing attendance")
			return
		}

		if len(u.Issues()) > 0 {
			a.Issues = append(a.Issues, &AttendanceIssue{
				Member: u,
				Reason: strings.Join(u.Issues(), ", "),
			})
			continue
		}

		a.Members = append(a.Members, u)
	}
}

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
		log.WithError(err).Error("responding to takeattendance command")
		return
	}
}

func TakeAttendanceCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user, err := members.Get(i.Member.User.ID)
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

	eventName := strings.TrimPrefix(data.Options[0].StringValue(), "(NEW) ")
	userIds := data.Options[1:]

	attendance := &Attendance{
		Id:     xid.New().String(),
		Name:   eventName,
		Issues: []*AttendanceIssue{},
	}

	channelId := config.GetString("DISCORD.CHANNELS.ATTENDANCE")

	usersList := []*members.Member{}
	for _, userId := range userIds {
		user, err := members.Get(userId.UserValue(s).ID)
		if err != nil {
			attendance.Issues = append(attendance.Issues, &AttendanceIssue{
				Member: &members.Member{
					Id:   userId.UserValue(s).ID,
					Name: userId.UserValue(s).Username,
				},
				Reason: "not in system",
			})
			continue
		}

		if len(user.Issues()) > 0 {
			attendance.Issues = append(attendance.Issues, &AttendanceIssue{
				Member: user,
				Reason: strings.Join(user.Issues(), ", "),
			})
			continue
		}

		usersList = append(usersList, user)
		// user.IncrementEventCount()
	}

	attendance.Members = usersList

	takeAttendanceComplete(s, i)

	var em *discordgo.Message
	em, _ = s.ChannelMessage(channelId, attendance.Name)

	if em == nil {
		// create a new attendance message
		em, err = s.ChannelMessageSend(channelId, fmt.Sprintf("%s (%s)", attendance.Name, xid.New().String()))
		if err != nil {
			log.WithError(err).Error("creating new attendance message")
			return
		}

		emThread, err := s.MessageThreadStart(em.ChannelID, em.ID, attendance.Name+" Attendance Thread", 1440)
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

	// we need to create a new message
	if len(etms) == 1 {
		// make the primary list
		_, err := s.ChannelMessageSendComplex(em.Thread.ID, &discordgo.MessageSend{
			Content: attendance.GenerateList(),
			Embeds:  []*discordgo.MessageEmbed{attendance.getIssuesEmbed()},
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
			log.WithError(err).Error("creating attendance thread message")
		}
		return
	}

	// we have a message already
	message := etms[len(etms)-2]
	currentUsersSplit := strings.Split(message.Content, "\n")
	currentUsersSplit = append(currentUsersSplit, strings.Split(message.Embeds[0].Fields[0].Value, "\n")...)
	for _, cu := range currentUsersSplit[1:] {
		if cu == "No members" || cu == "" {
			continue
		}
		uid := strings.ReplaceAll(cu, "<@", "")
		uid = strings.ReplaceAll(uid, ">", "")
		uid = strings.Split(uid, ":")[0]

		u, err := members.Get(uid)
		if err != nil {
			log.WithError(err).Error("getting user for existing attendance")
			return
		}
		attendance.Members = append(attendance.Members, u)
	}
	attendance.removeDuplicates()

	attendaceList := attendance.GenerateList()

	emb := []*discordgo.MessageEmbed{
		attendance.getIssuesEmbed(),
	}
	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: em.Thread.ID,
		ID:      message.ID,
		Content: &attendaceList,
		Embeds:  &emb,
	}); err != nil {
		log.WithError(err).Error("editing attendance thread message")
		return
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
	user, err := members.Get(i.Member.User.ID)
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

	// remove all members in usersList from currentUsersSplit
	message := etms[len(etms)-2]
	currentUsersSplit := strings.Split(message.Content, "\n")
	currentUsersSplit = append(currentUsersSplit, strings.Split(message.Embeds[0].Fields[0].Value, "\n")...)

	removedIds := ""
	for _, user := range userIds {
		removedIds += user.UserValue(s).ID + ","
	}

	attendance := &Attendance{}
	for _, userId := range currentUsersSplit[1:] {
		if userId == "No members" {
			continue
		}

		userId = strings.ReplaceAll(userId, "<@", "")
		userId = strings.ReplaceAll(userId, ">", "")
		userId = strings.Split(userId, ":")[0]

		if strings.Contains(removedIds, userId) {
			continue
		}

		user, err := members.Get(userId)
		if err != nil {
			log.WithError(err).Error("getting user")
			return
		}

		if len(user.Issues()) > 0 {
			attendance.Issues = append(attendance.Issues, &AttendanceIssue{
				Member: user,
				Reason: strings.Join(user.Issues(), ", "),
			})
			continue
		}

		attendance.Members = append(attendance.Members, user)
	}
	attendance.removeDuplicates()

	attendaceList := attendance.GenerateList()

	emb := []*discordgo.MessageEmbed{
		attendance.getIssuesEmbed(),
	}
	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: em.Thread.ID,
		ID:      message.ID,
		Content: &attendaceList,
		Embeds:  &emb,
	}); err != nil {
		log.WithError(err).Error("editing attendance thread message")
		return
	}

	removeAttendanceComplete(s, i)
}

func RecordAttendanceButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	threadId := i.Message.ChannelID

	threadMessages, err := s.ChannelMessages(threadId, 100, "", "", "")
	if err != nil {
		log.WithError(err).Error("getting attendance thread")
		return
	}

	attendance := &Attendance{}
	attendance.Parse(threadMessages)

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

func RecheckIssuesButtonHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	threadId := i.Message.ChannelID

	threadMessages, err := s.ChannelMessages(threadId, 100, "", "", "")
	if err != nil {
		log.WithError(err).Error("getting attendance thread")
		return
	}

	attendance := &Attendance{}
	attendance.Parse(threadMessages)

	usersList := []*members.Member{}
	for _, member := range attendance.Members {
		user, err := members.Get(member.Id)
		if err != nil {
			attendance.Issues = append(attendance.Issues, &AttendanceIssue{
				Member: &members.Member{
					Id:   member.Id,
					Name: member.Name,
				},
				Reason: "not in system",
			})
			continue
		}

		if len(user.Issues()) > 0 {
			attendance.Issues = append(attendance.Issues, &AttendanceIssue{
				Member: user,
				Reason: strings.Join(user.Issues(), ", "),
			})
			continue
		}

		usersList = append(usersList, user)
	}

	attendance.Members = usersList

	takeAttendanceComplete(s, i)

	etms, err := s.ChannelMessages(i.ChannelID, 100, "", "", "")
	if err != nil {
		log.WithError(err).Error("getting attendance thread messages")
		return
	}
	log.WithField("messages", etms).Debug("event message list")

	attendaceList := attendance.GenerateList()

	emb := []*discordgo.MessageEmbed{
		attendance.getIssuesEmbed(),
	}
	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: i.ChannelID,
		ID:      etms[len(etms)-2].ID,
		Content: &attendaceList,
		Embeds:  &emb,
	}); err != nil {
		log.WithError(err).Error("editing attendance thread message")
		return
	}
}
