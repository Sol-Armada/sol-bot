package attendance

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
)

func ListActive(limit int) ([]*Attendance, error) {
	cur, err := attendanceStore.List(bson.M{
		"$or": []bson.M{
			{"recorded": bson.M{"$eq": false}},
			{"status": bson.M{"$in": []Status{AttendanceStatusActive, AttendanceStatusReverted}}},
		},
	}, limit, 0)
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
	cur, err := attendanceStore.List(bson.M{
		"$or": []bson.M{
			{"recorded": bson.M{"$eq": false}},
			{"status": bson.M{"$eq": AttendanceStatusRecorded}},
		},
	}, limit, 0)
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

func List(filter any, limit int, page int) ([]*Attendance, error) {
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
