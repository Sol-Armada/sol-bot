package attendance

import (
	"github.com/sol-armada/sol-bot/members"
)

type Participant struct {
	Member         *members.Member `json:"member"`
	JoinedAtStart  bool            `json:"joined_at_start"`
	StayedUntilEnd bool            `json:"stayed_until_end"`
	IsManager      bool            `json:"is_manager"`
}

func (p *Participant) Save(attendanceID string) error {
	return attendanceStore.UpsertParticipant(attendanceID, p)
}

func (p *Participant) SetManager(attendanceID string) error {
	p.IsManager = true
	return p.Save(attendanceID)
}

func (p *Participant) SetStayedUntilEnd(attendanceID string, stayed bool) error {
	p.StayedUntilEnd = stayed
	return p.Save(attendanceID)
}

func (p *Participant) SetJoinedAtStart(attendanceID string, joined bool) error {
	p.JoinedAtStart = joined
	return p.Save(attendanceID)
}
