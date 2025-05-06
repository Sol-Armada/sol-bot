package attendance

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
)

func Watch(ctx context.Context, out chan Attendance) error {
	stream, err := attendanceStore.Watch(ctx, bson.D{})
	if err != nil {
		return err
	}
	defer stream.Close(ctx)

	for stream.Next(ctx) {
		var event bson.M
		if err := stream.Decode(&event); err != nil {
			return err
		}

		var attendance Attendance
		if err := bson.Unmarshal(event["fullDocument"].(bson.Raw), &attendance); err != nil {
			return err
		}

		out <- attendance
	}
	return ctx.Err()
}
