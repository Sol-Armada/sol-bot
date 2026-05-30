package tokens

import (
	"errors"
	"time"

	"github.com/rs/xid"
)

type Reason string

const (
	ReasonAttendance      Reason = "Attendance"
	ReasonAttendanceFull  Reason = "Stayed for full event"
	ReasonEventSuccessful Reason = "Event Successful"
	ReasonWonRaffle       Reason = "Won Raffle"
	ReasonManagerBonus    Reason = "Manager Bonus"
	ReasonOther           Reason = "Other"
)

type TokenRecord struct {
	Id           string    `json:"id" `
	MemberId     string    `json:"member_id" `
	Amount       int       `json:"amount" `
	Reason       Reason    `json:"reason" `
	AttendanceId *string   `json:"attendance_id" `
	Comment      *string   `json:"comment" `
	GiverId      *string   `json:"giver_id" `
	CreatedAt    time.Time `json:"created_at" `
}

var tokenStore tokenBackend

func Setup() error {
	return setupTokenBackend()
}

func New(memberId string, amount int, reason Reason, giverId, attendanceId, comment *string) *TokenRecord {
	return &TokenRecord{
		Id:           xid.New().String(),
		MemberId:     memberId,
		Amount:       amount,
		Reason:       reason,
		GiverId:      giverId,
		AttendanceId: attendanceId,
		Comment:      comment,
		CreatedAt:    time.Now(),
	}
}

func (d *TokenRecord) Save() error {
	if tokenStore == nil {
		return errors.New("token store not found")
	}
	return tokenStore.Insert(d)
}

func GetAllGrouped() (map[string][]TokenRecord, error) {
	if tokenStore == nil {
		return nil, errors.New("token store not found")
	}
	records, err := tokenStore.ListAll()
	if err != nil {
		return nil, err
	}
	grouped := map[string][]TokenRecord{}
	for _, r := range records {
		grouped[r.MemberId] = append(grouped[r.MemberId], r)
	}
	return grouped, nil
}

func GetByAttendanceId(attendanceId string) ([]TokenRecord, error) {
	if tokenStore == nil {
		return nil, errors.New("token store not found")
	}
	return tokenStore.ListByAttendanceID(attendanceId)
}

func GetByMemberIdAndAttendanceId(memberId, attendanceId string) ([]TokenRecord, error) {
	if tokenStore == nil {
		return nil, errors.New("token store not found")
	}
	return tokenStore.ListByMemberAndAttendance(memberId, attendanceId)
}

func GetBalanceByMemberId(memberId string) (int, error) {
	if tokenStore == nil {
		return 0, errors.New("token store not found")
	}
	balances, err := tokenStore.GetBalances()
	if err != nil {
		return 0, err
	}
	return balances[memberId], nil
}
