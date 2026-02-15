package attendance

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/xid"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/stores"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Status string

const (
	AttendanceStatusCreated  Status = "created"
	AttendanceStatusActive   Status = "active"
	AttendanceStatusRecorded Status = "recorded"
	AttendanceStatusReverted Status = "reverted"
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
	Successful  bool              `json:"successful" bson:"successful"`
	Active      bool              `json:"active" bson:"active"`
	Tokenable   bool              `json:"tokenable" bson:"tokenable"`
	Status      Status            `json:"status" bson:"status"`

	FromStart []string `json:"from_start" bson:"from_start"`
	Stayed    []string `json:"stayed" bson:"stayed"`

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

		Active: true,
		Status: AttendanceStatusActive,

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

func (a *Attendance) Record() error {
	a.Active = false
	a.Recorded = true
	a.Status = AttendanceStatusRecorded
	return a.Save()
}

func (a *Attendance) Revert() error {
	a.Recorded = false
	a.Status = AttendanceStatusReverted
	return a.Save()
}

func (a *Attendance) Save() error {
	if attendanceStore == nil {
		return errors.New("attendance store not found")
	}
	a.DateUpdated = time.Now().UTC()

	attendanceMap := map[string]any{}
	j, _ := json.Marshal(a)
	_ = json.Unmarshal(j, &attendanceMap)

	// convert members to just ids for mongo optimization
	memberIds := make([]string, len(a.Members))
	for i, member := range a.Members {
		memberIds[i] = member.Id
	}
	attendanceMap["members"] = memberIds

	// convert issues to just ids for mongo optimization
	issues, _ := attendanceMap["with_issues"].([]any)
	for i := range issues {
		issue, _ := issues[i].(map[string]any)
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
	a.Members = uniqueMembers(a.Members)
	a.WithIssues = uniqueMembers(a.WithIssues)
}

func uniqueMembers(mmbrs []*members.Member) []*members.Member {
	memberSet := make(map[string]*members.Member)
	for _, member := range mmbrs {
		if member == nil {
			continue
		}

		memberSet[member.Id] = member
	}

	uniqueList := make([]*members.Member, 0, len(memberSet))
	for _, member := range memberSet {
		uniqueList = append(uniqueList, member)
	}

	return uniqueList
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

func (a *Attendance) IsFromStart(m *members.Member) bool {
	for _, memberId := range a.FromStart {
		if memberId == m.Id {
			return true
		}
	}
	return false
}

func (a *Attendance) TheyStayed(m *members.Member) bool {
	for _, memberId := range a.Stayed {
		if memberId == m.Id {
			return true
		}
	}
	return false
}

func (a *Attendance) GetMember(id string) (*members.Member, bool) {
	for _, member := range a.Members {
		if member.Id == id {
			return member, true
		}
	}
	return nil, false
}
