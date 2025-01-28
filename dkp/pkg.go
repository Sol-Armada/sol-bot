package dkp

import (
	"context"
	"errors"
	"time"

	"github.com/rs/xid"
	"github.com/sol-armada/sol-bot/stores"
)

type Reason string

const (
	Attendance      Reason = "Attendance"
	AttendanceFull  Reason = "Stayed for full event"
	EventSuccessful Reason = "Event Successful"
	Other           Reason = "Other"
)

type DKP struct {
	Id           string    `json:"id" bson:"_id"`
	MemberId     string    `json:"member_id" bson:"member_id"`
	Amount       int       `json:"amount" bson:"amount"`
	Reason       Reason    `json:"reason" bson:"reason"`
	AttendanceId *string   `json:"attendance_id" bson:"attendance_id"`
	Comment      *string   `json:"comment" bson:"comment"`
	CreatedAt    time.Time `json:"created_at" bson:"created_at"`
}

var dkpStore *stores.DKPStore

func Setup() error {
	storesClient := stores.Get()
	ds, ok := storesClient.GetDKPStore()
	if !ok {
		return errors.New("members store not found")
	}
	dkpStore = ds

	return nil
}

func New(memberId string, amount int, reason Reason, attendanceId, comment *string) *DKP {
	return &DKP{
		Id:           xid.New().String(),
		MemberId:     memberId,
		Amount:       amount,
		Reason:       reason,
		AttendanceId: attendanceId,
		Comment:      comment,
		CreatedAt:    time.Now(),
	}
}

func (d *DKP) Save() error {
	return dkpStore.Insert(d)
}

func GetAll() ([]DKP, error) {
	cur, err := dkpStore.GetAll()
	if err != nil {
		return nil, err
	}

	var dkps []DKP
	for cur.Next(context.TODO()) {
		var d DKP
		if err := cur.Decode(&d); err != nil {
			return nil, err
		}
		dkps = append(dkps, d)
	}

	return dkps, nil
}

func GetBalanceByMemberId(memberId string) (int, error) {
	cur, err := dkpStore.GetAllBalances()
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
