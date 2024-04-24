package attendance

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/xid"
	"github.com/sol-armada/admin/members"
	"github.com/sol-armada/admin/stores"
	"go.mongodb.org/mongo-driver/bson"
)

type AttendanceIssue struct {
	Member *members.Member `json:"member"`
	Reason string          `json:"reason"`
}

type Attendance struct {
	Id       string             `json:"id" bson:"_id"`
	Name     string             `json:"name"`
	Members  []*members.Member  `json:"members"`
	Issues   []*AttendanceIssue `json:"issues"`
	Recorded bool               `json:"recorded"`

	ChannelId string `json:"channel_id" bson:"channel_id"`
	MessageId string `json:"message_id" bson:"message_id"`

	DateCreated time.Time `json:"date_created" bson:"date_created"`
	DateUpdated time.Time `json:"date_updated" bson:"date_updated"`
}

var attendanceStore *stores.AttendanceStore

func Setup() error {
	storesClient := stores.Get()
	as, ok := storesClient.GetAttendanceStore()
	if !ok {
		return errors.New("attendance store not found")
	}
	attendanceStore = as
	return nil
}

func New(name string) (*Attendance, error) {
	attendance := &Attendance{
		Id:          xid.New().String(),
		Name:        name,
		DateCreated: time.Now().UTC(),
		DateUpdated: time.Now().UTC(),
	}

	if err := attendanceStore.Upsert(attendance.Id, attendance); err != nil {
		return nil, err
	}

	return attendance, nil
}

func Get(id string) (*Attendance, error) {
	cur, err := attendanceStore.Get(id)
	if err != nil {
		return nil, err
	}

	attendance := &Attendance{}

	for cur.Next(context.TODO()) {
		if err := cur.Decode(attendance); err != nil {
			return nil, err
		}
	}

	return attendance, nil
}

func GetFromMessage(message *discordgo.Message) (*Attendance, error) {
	// get the Id from the footer of the embed
	// Last Updated 00-00-00T00:00:00Z (1234567890)
	reg := regexp.MustCompile(`Last Updated .*?\((.*?)\)`)

	attendanceId := reg.FindStringSubmatch(message.Embeds[0].Footer.Text)[1]
	cur, err := attendanceStore.Get(attendanceId)
	if err != nil {
		return nil, err
	}

	attendance := &Attendance{}
	if err := cur.Decode(attendance); err != nil {
		return nil, err
	}

	return attendance, nil
}

func NewFromThreadMessages(threadMessages []*discordgo.Message) (*Attendance, error) {
	mainMessage := threadMessages[len(threadMessages)-1].ReferencedMessage
	attendanceMessage := threadMessages[len(threadMessages)-2]

	attendance := &Attendance{}

	// get the ID between ( )
	reg := regexp.MustCompile(`(.*?)\((.*?)\)`)
	attendance.Id = reg.FindStringSubmatch(mainMessage.Content)[1]

	// get the name before ( )
	attendance.Name = reg.FindStringSubmatch(mainMessage.Content)[0]

	currentUsersSplit := strings.Split(attendanceMessage.Content, "\n")
	currentUsersSplit = append(currentUsersSplit, strings.Split(attendanceMessage.Embeds[0].Fields[0].Value, "\n")...)
	for _, cu := range currentUsersSplit[1:] {
		if cu == "No members" || cu == "" {
			continue
		}
		memberid := strings.ReplaceAll(cu, "<@", "")
		memberid = strings.ReplaceAll(memberid, ">", "")
		memberid = strings.Split(memberid, ":")[0]

		member, err := members.Get(memberid)
		if err != nil {
			return nil, err
		}

		attendance.AddMember(member)
	}

	return attendance, nil
}

func ListActive(limit int64) ([]*Attendance, error) {
	cur, err := attendanceStore.List(bson.M{"recorded": bson.M{"$eq": false}}, limit)
	if err != nil {
		return nil, err
	}

	var attendances []*Attendance

	for cur.Next(context.TODO()) {
		attendance := &Attendance{}
		if err := cur.Decode(attendance); err != nil {
			return nil, err
		}
		attendances = append(attendances, attendance)
	}

	return attendances, nil
}

func GetMemberAttendanceCount(id string) int {
	res, err := attendanceStore.List(bson.M{"members": bson.M{"$elemMatch": bson.M{"_id": id}}}, 0)
	if err != nil {
		return 0
	}

	return int(res.RemainingBatchLength())
}

func (a *Attendance) AddMember(member *members.Member) {
	defer a.removeDuplicates()

	memberIssues := Issues(member)
	if len(memberIssues) == 0 {
		a.Issues = append(a.Issues, &AttendanceIssue{
			Member: member,
			Reason: strings.Join(memberIssues, ", "),
		})
		return
	}

	a.Members = append(a.Members, member)
}

func (a *Attendance) RemoveMember(member *members.Member) {
	for i, m := range a.Members {
		if m == member {
			a.Members = append(a.Members[:i], a.Members[i+1:]...)
			break
		}
	}

	for i, m := range a.Issues {
		if m.Member == member {
			a.Issues = append(a.Issues[:i], a.Issues[i+1:]...)
			break
		}
	}

	a.removeDuplicates()
}

func (a *Attendance) RecheckIssues() error {
	newIssues := []*AttendanceIssue{}
	for _, issue := range a.Issues {
		memberIssues := Issues(issue.Member)
		if len(memberIssues) != 0 {
			newIssues = append(newIssues, &AttendanceIssue{
				Member: issue.Member,
				Reason: strings.Join(memberIssues, ", "),
			})
		}
	}
	a.Issues = newIssues

	a.removeDuplicates()

	return a.Save()
}

func (a *Attendance) ToDiscordMessage() *discordgo.MessageSend {
	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Attendees",
			Value:  "",
			Inline: true,
		},
	}

	i := 0
	for _, member := range a.Members {
		field := fields[len(fields)-1]
		field.Value += "<@" + member.Id + ">"

		// if not the 10th, add a new line
		if i%10 != 9 {
			field.Value += "\n"
		}

		// for every 10 members, make a new field
		if i%10 == 0 && i != 0 {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "",
				Value:  "",
				Inline: true,
			})
		}
		i++
	}

	if len(a.Issues) > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Attendees with Issues",
			Value:  "",
			Inline: true,
		})

		i = 0
		for _, issue := range a.Issues {
			field := fields[len(fields)-1]

			field.Value += "<@" + issue.Member.Id + "> - " + issue.Reason

			// if not the 10th, add a new line
			if i%10 != 9 {
				field.Value += "\n"
			}

			// for every 10 members, make a new field
			if i%10 == 0 && i != 0 {
				fields = append(fields, &discordgo.MessageEmbedField{
					Name:   "Attendees with Issues (continued)",
					Value:  "",
					Inline: true,
				})
			}
			i++
		}
	}

	embeds := []*discordgo.MessageEmbed{
		{
			Title:       a.Name,
			Description: a.Id,
			Timestamp:   a.DateCreated.Format(time.RFC3339),
			Fields:      fields,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Last Updated " + a.DateUpdated.Format(time.RFC3339),
			},
		},
	}

	return &discordgo.MessageSend{
		Embeds: embeds,
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Record",
						Style:    discordgo.SuccessButton,
						Disabled: a.Recorded,
						Emoji: &discordgo.ComponentEmoji{
							Name: "✅",
						},
						CustomID: "attendance:record:" + a.Id,
					},
					discordgo.Button{
						Label:    "Delete",
						Style:    discordgo.DangerButton,
						Disabled: a.Recorded,
						Emoji: &discordgo.ComponentEmoji{
							Name: "🗑️",
						},
						CustomID: "attendance:delete:" + a.Id,
					},
					discordgo.Button{
						Label:    "Recheck Issues",
						Style:    discordgo.PrimaryButton,
						Disabled: a.Recorded,
						Emoji: &discordgo.ComponentEmoji{
							Name: "🔁",
						},
						CustomID: "attendance:recheck:" + a.Id,
					},
				},
			},
		},
	}
}

func (a *Attendance) Record() error {
	a.Recorded = true
	return a.Save()
}

func (a *Attendance) Save() error {
	if attendanceStore == nil {
		return errors.New("attendance store not found")
	}
	a.DateUpdated = time.Now().UTC()

	attendanceMap := map[string]interface{}{}
	j, _ := json.Marshal(a)
	_ = json.Unmarshal(j, &attendanceMap)

	memberIds := []string{}
	for _, member := range a.Members {
		memberIds = append(memberIds, member.Id)
	}
	attendanceMap["members"] = memberIds

	issues := attendanceMap["issues"].([]interface{})
	for i, issue := range issues {
		issue := issue.(map[string]interface{})
		issue["member"] = issue["member"].(map[string]interface{})["id"].(string)
		issues[i] = issue
	}
	attendanceMap["issues"] = issues

	return attendanceStore.Upsert(a.Id, attendanceMap)
}

func (a *Attendance) removeDuplicates() {
	list := []*members.Member{}
	for _, member := range a.Members {
		if slices.Contains(list, member) {
			continue
		}
		list = append(list, member)
	}
	a.Members = list

	issuesList := []*AttendanceIssue{}
	for _, memberId := range a.Issues {
		found := false
		for _, v := range issuesList {
			if memberId.Member == v.Member {
				found = true
				break
			}
		}
		if found {
			continue
		}
		issuesList = append(issuesList, memberId)
	}
	a.Issues = issuesList
}

// func (a *Attendance) GenerateList() string {
// 	// remove duplicates
// 	list := make(map[string]*members.Member)
// 	for _, u := range a.Members {
// 		list[u.Id] = u
// 	}

// 	a.Members = []*members.Member{}
// 	for _, u := range list {
// 		a.Members = append(a.Members, u)
// 	}

// 	slices.SortFunc(a.Members, func(a, b *members.Member) int {
// 		if a.Rank > b.Rank {
// 			return 1
// 		}
// 		if a.Rank < b.Rank {
// 			return -1
// 		}
// 		if a.Name < b.Name {
// 			return 1
// 		}
// 		if a.Name > b.Name {
// 			return -1
// 		}

// 		return 0
// 	})

// 	m := ""
// 	for i, u := range a.Members {
// 		m += fmt.Sprintf("<@%s>", u.Id)
// 		if i < len(a.Members)-1 {
// 			m += "\n"
// 		}
// 	}

// 	if m == "" {
// 		m = "No members"
// 	}

// 	return "Attendance List:\n" + m
// }

// func (a *Attendance) GetIssuesEmbed() *discordgo.MessageEmbed {
// 	embed := &discordgo.MessageEmbed{
// 		Title:       "Users with Issues",
// 		Description: "List of members with attendance credit issues",
// 		Fields:      []*discordgo.MessageEmbedField{},
// 	}

// 	fieldValue := ""
// 	for _, issue := range a.Issues {
// 		fieldValue += fmt.Sprintf("<@%s>: %s\n", issue.Member.Id, issue.Reason)
// 	}
// 	field := &discordgo.MessageEmbedField{
// 		Name:  "Member - Issues",
// 		Value: fieldValue,
// 	}
// 	embed.Fields = append(embed.Fields, field)

// 	return embed
// }
