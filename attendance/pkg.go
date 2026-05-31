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

type Attendance struct {
	Id          string          `json:"id" `
	Name        string          `json:"name"`
	SubmittedBy *members.Member `json:"submitted_by" `
	Recorded    bool            `json:"recorded"`
	Successful  bool            `json:"successful" `
	Tokenable   bool            `json:"tokenable" `
	Status      Status          `json:"status" `

	ChannelId string `json:"channel_id" `
	MessageId string `json:"message_id" `

	DateCreated time.Time `json:"date_created" `
	DateUpdated time.Time `json:"date_updated" `
}

var (
	ErrAttendanceNotFound = errors.New("attendance not found")
)

var attendanceStore attendanceBackend

func Setup() error {
	return setupAttendanceBackend()
}

func New(name string, submittedBy *members.Member) (*Attendance, error) {
	attendance := &Attendance{
		Id:          xid.New().String(),
		Name:        name,
		DateCreated: time.Now().UTC(),
		DateUpdated: time.Now().UTC(),
		SubmittedBy: submittedBy,

		Status: AttendanceStatusCreated,

		ChannelId: settings.GetString("FEATURES.ATTENDANCE.CHANNEL_ID"),
	}

	return attendance, attendance.Save()
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

		if err := attendance.AddParticipant(member); err != nil {
			return nil, err
		}
	}

	return attendance, nil
}

func GetMemberAttendanceCount(memberId string) (int, error) {
	return attendanceStore.GetCount(memberId)
}

func (a *Attendance) Record() error {
	a.Recorded = true
	a.Status = AttendanceStatusRecorded
	return a.Save()
}

func (a *Attendance) Revert() error {
	a.Recorded = false
	a.Status = AttendanceStatusReverted
	return a.Save()
}

// func (a *Attendance) AddMember(member *members.Member) error {
// 	return attendanceStore.CreateParticipant(a.Id, &Participant{
// 		Member: member,
// 	})
// }

func (a *Attendance) Save() error {
	if attendanceStore == nil {
		return errors.New("attendance store not found")
	}
	a.DateUpdated = time.Now().UTC()
	return attendanceStore.Upsert(a)
}

func (a *Attendance) Delete() error {
	return attendanceStore.Delete(a.Id)
}

func (a *Attendance) Participants() ([]*Participant, error) {
	return attendanceStore.ListParticipants(a.Id)
}

func (a *Attendance) Members() ([]*members.Member, error) {
	return attendanceStore.ListMembers(a.Id)
}

func (a *Attendance) WithIssues() ([]*members.Member, error) {
	mbrs, err := a.Members()
	if err != nil {
		return nil, err
	}

	withIssues := []*members.Member{}
	for _, member := range mbrs {
		if member == nil {
			continue
		}

		if len(Issues(member)) > 0 {
			withIssues = append(withIssues, member)
		}
	}

	return withIssues, nil
}

func (a *Attendance) HasParticipant(memberId string) (bool, error) {
	participants, err := a.Participants()
	if err != nil {
		return false, err
	}

	for _, participant := range participants {
		if participant.Member.Id == memberId {
			return true, nil
		}
	}

	return false, nil
}

func (a *Attendance) PraticipantsFromStart() ([]*Participant, error) {
	participants, err := a.Participants()
	if err != nil {
		return nil, err
	}

	fromStart := []*Participant{}
	for _, participant := range participants {
		if participant.JoinedAtStart {
			fromStart = append(fromStart, participant)
		}
	}

	return fromStart, nil
}

func (a *Attendance) GetParticipant(memberId string) (*Participant, error) {
	participants, err := a.Participants()
	if err != nil {
		return nil, err
	}

	for _, participant := range participants {
		if participant.Member.Id == memberId {
			return participant, nil
		}
	}

	return nil, nil
}

func (a *Attendance) RecheckIssues() error {
	participants, err := a.Participants()
	if err != nil {
		return err
	}

	for _, participant := range participants {
		if participant.Member == nil {
			continue
		}

		if err := participant.Member.UpdateRsiInfo(); err != nil {
			return err
		}
	}

	return nil
}

func (a *Attendance) AddParticipant(member *members.Member) error {
	return attendanceStore.CreateParticipant(a.Id, &Participant{
		Member: member,
	})
}

func (a *Attendance) RemoveParticipant(member *members.Member) error {
	return attendanceStore.RemoveParticipant(a.Id, member.Id)
}

func (a *Attendance) SetStatus(status Status) error {
	a.Status = status
	return a.Save()
}

func (a *Attendance) SetParticipantManager(memberId string) error {
	participant, err := a.GetParticipant(memberId)
	if err != nil {
		return err
	}

	return participant.SetManager(a.Id)
}

func (a *Attendance) SetParticipantStayedUntilEnd(memberId string) error {
	participant, err := a.GetParticipant(memberId)
	if err != nil {
		return err
	}

	return participant.SetStayedUntilEnd(a.Id, true)
}
