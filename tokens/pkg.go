package tokens

import (
	"context"
	"errors"
	"time"

	"github.com/rs/xid"
	"github.com/sol-armada/sol-bot/stores"
)

type Reason string

const (
	ReasonAttendance      Reason = "Attendance"
	ReasonAttendanceFull  Reason = "Stayed for full event"
	ReasonEventSuccessful Reason = "Event Successful"
	ReasonOther           Reason = "Other"
)

type TokenRecord struct {
	Id           string    `json:"id" bson:"_id"`
	MemberId     string    `json:"member_id" bson:"member_id"`
	Amount       int       `json:"amount" bson:"amount"`
	Reason       Reason    `json:"reason" bson:"reason"`
	AttendanceId *string   `json:"attendance_id" bson:"attendance_id"`
	Comment      *string   `json:"comment" bson:"comment"`
	GiverId      *string   `json:"giver_id" bson:"giver_id"`
	CreatedAt    time.Time `json:"created_at" bson:"created_at"`
}

var tokenStore *stores.TokenStore

func Setup() error {
	storesClient := stores.Get()
	ts, ok := storesClient.GetTokensStore()
	if !ok {
		return errors.New("token store not found")
	}
	tokenStore = ts

	return nil
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
	return tokenStore.Insert(d)
}

func GetAllGrouped() (map[string][]TokenRecord, error) {
	cur, err := tokenStore.GetAll()
	if err != nil {
		return nil, err
	}

	type GroupedRecord struct {
		Id           string        `json:"id" bson:"_id"`
		TokenRecords []TokenRecord `json:"token_records" bson:"token_records"`
	}

	var groupedRecords []GroupedRecord
	for cur.Next(context.TODO()) {
		var d GroupedRecord
		if err := cur.Decode(&d); err != nil {
			return nil, err
		}
		groupedRecords = append(groupedRecords, d)
	}

	tokenRecords := make(map[string][]TokenRecord, len(groupedRecords))
	for _, r := range groupedRecords {
		tokenRecords[r.Id] = append(tokenRecords[r.Id], r.TokenRecords...)
	}

	return tokenRecords, nil
}

func GetBalanceByMemberId(memberId string) (int, error) {
	cur, err := tokenStore.GetAllBalances()
	if err != nil {
		return 0, err
	}

	var balance int
	for cur.Next(context.TODO()) {
		var result struct {
			Id      string `bson:"_id"`
			Balance int    `bson:"balance"`
		}

		if err := cur.Decode(&result); err != nil {
			return 0, err
		}

		if result.Id == memberId {
			balance = result.Balance
			break
		}
	}

	return balance, nil
}
