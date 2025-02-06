package attendance

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func Watch(ctx context.Context, out chan Attendance) error {
	lastRecordTS := time.Now()

	for {
		select {
		case <-ctx.Done():
			close(out)
			return nil
		default:
		}

		cur, err := attendanceStore.List(bson.D{
			{Key: "date_created", Value: bson.D{
				{Key: "$gt", Value: lastRecordTS},
			}},
		}, 0, 0)
		if err != nil {
			return err
		}

		for cur.Next(ctx) {
			var d Attendance
			if err := cur.Decode(&d); err != nil {
				return err
			}

			out <- d
			lastRecordTS = d.DateCreated
		}

		time.Sleep(1 * time.Second)
	}
}
