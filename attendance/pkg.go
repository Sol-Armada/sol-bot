package attendance

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/xid"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/settings"
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

type Participant struct {
	Member          *members.Member `json:"member"`
	JoinedAtStart   bool            `json:"joined_at_start"`
	StayedUntilEnd  bool            `json:"stayed_until_end"`
	HasIssue        bool            `json:"has_issue"`
}

type Attendance struct {
	Id          string            `json:"id" bson:"_id"`
	Name        string            `json:"name"`
	SubmittedBy *members.Member   `json:"submitted_by" bson:"submitted_by"`
	Members     []*members.Member `json:"members"`
	Participants []Participant    `json:"participants"`
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

var attendanceStore attendanceBackend

func Setup() error {
	return setupAttendanceBackend()
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
	if attendanceStore == nil {
		return nil, errors.New("attendance store not found")
	}
	attendance, err := attendanceStore.Get(id)
	if err != nil {
		return nil, err
	}
	if attendance == nil || attendance.Id == "" {
		return nil, ErrAttendanceNotFound
	}
	return attendance, nil
}

func GetFromMessage(message *discordgo.Message) (*Attendance, error) {
	// get the Id from the footer of the embed
	// Last Updated 00-00-00T00:00:00Z (1234567890)
	reg := regexp.MustCompile(`Last Updated .*?\((.*?)\)`)
	attendanceID := reg.FindStringSubmatch(message.Embeds[0].Footer.Text)[1]
	return Get(attendanceID)
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
	attendances, err := List(nil, 0, 0)
	if err != nil {
		return nil, err
	}

	for _, attendance := range attendances {
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
	return attendanceStore.Upsert(a)
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
