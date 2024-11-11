package attendance

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/xid"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/ranks"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/stores"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Payouts struct {
	Total     int64 `json:"total"`
	PerMember int64 `json:"per_member"`
	OrgTake   int64 `json:"org_take"`
}

type Attendance struct {
	Id          string            `json:"id" bson:"_id"`
	Name        string            `json:"name"`
	SubmittedBy *members.Member   `json:"submitted_by" bson:"submitted_by"`
	Members     []*members.Member `json:"members"`
	WithIssues  []*members.Member `json:"with_issues" bson:"with_issues"`
	Recorded    bool              `json:"recorded"`
	Payouts     *Payouts          `json:"payouts" bson:"payouts"`

	ChannelId string `json:"channel_id" bson:"channel_id"`
	MessageId string `json:"message_id" bson:"message_id"`

	DateCreated time.Time `json:"date_created" bson:"date_created"`
	DateUpdated time.Time `json:"date_updated" bson:"date_updated"`
}

var (
	ErrAttendanceNotFound = errors.New("attendance not found")
)

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

func New(name string, submittedBy *members.Member) *Attendance {
	attendance := &Attendance{
		Id:          xid.New().String(),
		Name:        name,
		DateCreated: time.Now().UTC(),
		DateUpdated: time.Now().UTC(),
		SubmittedBy: submittedBy,

		ChannelId: settings.GetString("FEATURES.ATTENDANCE.CHANNEL_ID"),
	}

	return attendance
}

func Get(id string) (*Attendance, error) {
	cur, err := attendanceStore.Get(id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrAttendanceNotFound
		}

		return nil, err
	}

	attendance := &Attendance{}

	for cur.Next(context.TODO()) {
		if err := cur.Decode(attendance); err != nil {
			return nil, err
		}
	}

	if attendance.Id == "" {
		return nil, ErrAttendanceNotFound
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

func ListActive(limit int) ([]*Attendance, error) {
	cur, err := attendanceStore.List(bson.M{"recorded": bson.M{"$eq": false}}, limit, 0)
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

func ListRecorded(limit int) ([]*Attendance, error) {
	cur, err := attendanceStore.List(bson.M{"recorded": bson.M{"$eq": true}}, limit, 0)
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

func List(filter interface{}, limit int, page int) ([]*Attendance, error) {
	cur, err := attendanceStore.List(filter, limit, page)
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

func GetMemberAttendanceCount(memberId string) (int, error) {
	return attendanceStore.GetCount(memberId)
}

func GetMemberAttendanceRecords(memberId string) ([]*Attendance, error) {
	records := []*Attendance{}

	cur, err := attendanceStore.List(bson.D{}, 0, 0)
	if err != nil {
		return nil, err
	}

	for cur.Next(context.TODO()) {
		attendance := &Attendance{}
		if err := cur.Decode(attendance); err != nil {
			return nil, err
		}

		for _, member := range attendance.Members {
			if member.Id == memberId {
				records = append(records, attendance)
				break
			}
		}
	}

	return records, nil
}

func (a *Attendance) AddMember(member *members.Member) {
	defer a.removeDuplicates()

	memberIssues := Issues(member)
	if len(memberIssues) > 0 {
		a.WithIssues = append(a.WithIssues, member)
		return
	}

	a.Members = append(a.Members, member)
}

func (a *Attendance) RemoveMember(member *members.Member) {
	for i, m := range a.Members {
		if m.Id == member.Id {
			a.Members = append(a.Members[:i], a.Members[i+1:]...)
			break
		}
	}

	for i, m := range a.WithIssues {
		if m.Id == member.Id {
			a.WithIssues = append(a.WithIssues[:i], a.WithIssues[i+1:]...)
			break
		}
	}

	a.removeDuplicates()
}

func (a *Attendance) RecheckIssues() error {
	attendees := []*members.Member{}
	for _, member := range a.Members {
		issues := Issues(member)

		if len(issues) == 0 {
			attendees = append(attendees, member)
		} else {
			a.WithIssues = append(a.WithIssues, member)
		}
	}
	a.Members = attendees

	newIssues := []*members.Member{}
	for _, member := range a.WithIssues {
		memberIssues := Issues(member)
		if len(memberIssues) != 0 {
			newIssues = append(newIssues, member)
		} else {
			a.Members = append(a.Members, member)
		}
	}
	a.WithIssues = newIssues

	a.removeDuplicates()

	return a.Save()
}

func (a *Attendance) ToDiscordMessage() *discordgo.MessageSend {
	fields := []*discordgo.MessageEmbedField{
		{
			Name:  "Submitted By",
			Value: "<@" + a.SubmittedBy.Id + ">",
		},
		{
			Name:   "Attendees",
			Value:  "No Attendees",
			Inline: true,
		},
	}

	if len(a.Members) > 0 {
		sort.Slice(a.Members, func(i, j int) bool {
			if a.Members[i].IsGuest {
				return false
			}
			if a.Members[i].IsAffiliate {
				return false
			}

			if a.Members[i].IsAlly {
				return false
			}

			if a.Members[j].IsGuest {
				return true
			}

			if a.Members[j].IsAffiliate {
				return true
			}

			if a.Members[j].IsAlly {
				return true
			}

			if a.Members[i].Rank < a.Members[j].Rank {
				return true
			}

			if a.Members[i].Rank != ranks.None && a.Members[i].Rank == a.Members[j].Rank {
				return a.Members[i].Name < a.Members[j].Name
			}

			return false
		})

		fields[1].Value = ""

		i := 0
		for _, member := range a.Members {
			// for every 10 members, make a new field
			if i%10 == 0 && i != 0 {
				fields = append(fields, &discordgo.MessageEmbedField{
					Name:   "Attendees (continued)",
					Value:  "",
					Inline: true,
				})
			}

			field := fields[len(fields)-1]
			field.Value += "<@" + member.Id + ">"

			// if not the 10th, add a new line
			if i%10 != 9 {
				field.Value += "\n"
			}
			i++
		}
	}

	if len(a.WithIssues) > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Attendees with Issues",
			Value:  "",
			Inline: true,
		})

		i := 0
		for _, member := range a.WithIssues {
			field := fields[len(fields)-1]

			field.Value += "<@" + member.Id + "> - " + strings.Join(Issues(member), ", ")

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

	if a.Payouts != nil {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Payouts",
			Value:  "Total: " + strconv.Itoa(int(a.Payouts.Total)) + "\nPer Member: " + strconv.Itoa(int(a.Payouts.PerMember)) + "\nOrg Take: " + strconv.Itoa(int(a.Payouts.OrgTake)),
			Inline: false,
		})
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

	components := []discordgo.MessageComponent{}
	if !a.Recorded {
		components = []discordgo.MessageComponent{
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
					discordgo.Button{
						Label:    "Add Payout",
						Style:    discordgo.PrimaryButton,
						Disabled: a.Recorded,
						Emoji: &discordgo.ComponentEmoji{
							Name: "💰",
						},
						CustomID: "attendance:payout:" + a.Id,
					},
				},
			},
		}

		if a.Payouts != nil {
			components[0].(discordgo.ActionsRow).Components[3] = discordgo.Button{
				Label:    "Edit Payout",
				Style:    discordgo.PrimaryButton,
				Disabled: a.Recorded,
				Emoji: &discordgo.ComponentEmoji{
					Name: "💰",
				},
				CustomID: "attendance:payout:" + a.Id,
			}
		}
	}

	return &discordgo.MessageSend{
		Embeds:     embeds,
		Components: components,
	}
}

func (a *Attendance) Record() error {
	a.Recorded = true
	return a.Save()
}

func (a *Attendance) Revert() error {
	a.Recorded = false
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

	// convert members to just ids for mongo optimization
	memberIds := make([]string, len(a.Members))
	for i, member := range a.Members {
		memberIds[i] = member.Id
	}
	attendanceMap["members"] = memberIds

	// convert issues to just ids for mongo optimization
	issues, _ := attendanceMap["with_issues"].([]interface{})
	for i := range issues {
		issue, _ := issues[i].(map[string]interface{})
		issues[i] = issue["id"]
	}
	attendanceMap["with_issues"] = issues

	// convert submitted by to just id for mongo optimization
	attendanceMap["submitted_by"] = a.SubmittedBy.Id

	// convert date_created to mongo datetime
	attendanceMap["date_created"] = a.DateCreated.UTC()

	// convert date_updated to mongo datetime
	attendanceMap["date_updated"] = a.DateUpdated.UTC()

	return attendanceStore.Upsert(a.Id, attendanceMap)
}

func (a *Attendance) removeDuplicates() {
	memberSet := map[string]*members.Member{}
	for _, member := range a.Members {
		memberSet[member.Id] = member
	}
	a.Members = []*members.Member{}
	for _, member := range memberSet {
		a.Members = append(a.Members, member)
	}

	withIssuesSet := map[string]*members.Member{}
	for _, member := range a.WithIssues {
		withIssuesSet[member.Id] = member
	}
	a.WithIssues = []*members.Member{}
	for _, member := range withIssuesSet {
		a.WithIssues = append(a.WithIssues, member)
	}
}

func (a *Attendance) Delete() error {
	return attendanceStore.Delete(a.Id)
}

func (a *Attendance) AddPayout(total, perMember, orgTake int64) error {
	a.Payouts = &Payouts{
		Total:     total,
		PerMember: perMember,
		OrgTake:   orgTake,
	}

	return a.Save()
}
