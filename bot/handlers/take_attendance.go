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
	"github.com/sol-armada/admin/utils"
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

		member, err := members.Get(uid)
		if err != nil {
			log.WithError(err).Error("getting user for existing attendance")
			return
		}

		if len(issues(member)) > 0 {
			a.Issues = append(a.Issues, &AttendanceIssue{
				Member: member,
				Reason: strings.Join(issues(member), ", "),
			})
			continue
		}

		a.Members = append(a.Members, member)
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
	if !utils.StringSliceContainsOneOf(i.Member.Roles, config.GetStringSlice("ATTENDANCE.ALLOWED_ROLES")) {
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
	memberIds := data.Options[1:]

	attendance := &Attendance{
		Id:     xid.New().String(),
		Name:   eventName,
		Issues: []*AttendanceIssue{},
	}

	channelId := config.GetString("FEATURES.ATTENDANCE.CHANNEL_ID")

	membersList := []*members.Member{}
	for _, memberId := range memberIds {
		member, err := members.Get(memberId.UserValue(s).ID)
		if err != nil {
			attendance.Issues = append(attendance.Issues, &AttendanceIssue{
				Member: &members.Member{
					Id:   memberId.UserValue(s).ID,
					Name: memberId.UserValue(s).Username,
				},
				Reason: "not in system",
			})
			continue
		}

		if len(issues(member)) > 0 {
			attendance.Issues = append(attendance.Issues, &AttendanceIssue{
				Member: member,
				Reason: strings.Join(issues(member), ", "),
			})
			continue
		}

		membersList = append(membersList, member)
	}

	attendance.Members = membersList

	takeAttendanceComplete(s, i)

	var eventMessage *discordgo.Message
	eventMessage, _ = s.ChannelMessage(channelId, attendance.Name)

	if eventMessage == nil {
		// create a new attendance message
		eventMessage, err := s.ChannelMessageSend(channelId, fmt.Sprintf("%s (%s)", attendance.Name, xid.New().String()))
		if err != nil {
			log.WithError(err).Error("creating new attendance message")
			return
		}

		emThread, err := s.MessageThreadStart(eventMessage.ChannelID, eventMessage.ID, attendance.Name+" Attendance Thread", 1440)
		if err != nil {
			log.WithError(err).Error("creating new attendance message")
			return
		}

		eventMessage.Thread = emThread
	}

	eventThreadMessages, err := s.ChannelMessages(eventMessage.Thread.ID, 100, "", "", "")
	if err != nil {
		log.WithError(err).Error("getting attendance thread messages")
		return
	}
	log.WithField("messages", eventThreadMessages).Debug("event message list")

	// we need to create a new message
	if len(eventThreadMessages) == 1 {
		// make the primary list
		_, err := s.ChannelMessageSendComplex(eventMessage.Thread.ID, &discordgo.MessageSend{
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
			log.WithError(err).Error("getting member for existing attendance")
			return
		}
		attendance.Members = append(attendance.Members, member)
	}
	attendance.removeDuplicates()

	attendaceList := attendance.GenerateList()

	emb := []*discordgo.MessageEmbed{
		attendance.getIssuesEmbed(),
	}
	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: eventMessage.Thread.ID,
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
		channelMessages, err := s.ChannelMessages(config.GetString("FEATURES.ATTENDANCE.CHANNEL_ID"), 5, "", "", "")
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
		log.WithError(err).Error("responding to removeattendance autocomplete")
		return
	}
}

func RemoveAttendanceCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !utils.StringSliceContainsOneOf(i.Member.Roles, config.GetStringSlice("ATTENDANCE.ALLOWED_ROLES")) {
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
			log.WithError(err).Error("getting user")
			return
		}

		if len(issues(member)) > 0 {
			attendance.Issues = append(attendance.Issues, &AttendanceIssue{
				Member: member,
				Reason: strings.Join(issues(member), ", "),
			})
			continue
		}

		attendance.Members = append(attendance.Members, member)
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

	membersList := []*members.Member{}
	for _, aMember := range attendance.Members {
		member, err := members.Get(aMember.Id)
		if err != nil {
			attendance.Issues = append(attendance.Issues, &AttendanceIssue{
				Member: &members.Member{
					Id:   aMember.Id,
					Name: aMember.Name,
				},
				Reason: "not in system",
			})
			continue
		}

		if len(issues(member)) > 0 {
			attendance.Issues = append(attendance.Issues, &AttendanceIssue{
				Member: member,
				Reason: strings.Join(issues(member), ", "),
			})
			continue
		}

		membersList = append(membersList, member)
	}

	attendance.Members = membersList

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

func issues(m *members.Member) []string {
	issues := []string{}

	if m.IsBot {
		issues = append(issues, "bot")
	}

	if m.Rank == ranks.Guest {
		issues = append(issues, "guest")
	}

	if m.Rank == ranks.Recruit && !m.RSIMember {
		issues = append(issues, "non-rsi member but is recruit")
	}

	if m.IsAlly {
		issues = append(issues, "ally")
	}

	if m.BadAffiliation {
		issues = append(issues, "bad affiliation")
	}

	if m.PrimaryOrg == "REDACTED" {
		issues = append(issues, "redacted org")
	}

	if m.Rank <= ranks.Member && m.PrimaryOrg != config.GetString("rsi_org_sid") {
		issues = append(issues, "bad primary org")
	}

	switch m.Rank {
	case ranks.Recruit:
		if m.Events >= 3 {
			issues = append(issues, "max event credits for this rank (3)")
		}
	case ranks.Member:
		if m.Events >= 10 {
			issues = append(issues, "max event credits for this rank (10)")
		}
	case ranks.Technician:
		if m.Events >= 20 {
			issues = append(issues, "max event credits for this rank (20)")
		}
	}

	return issues
}
