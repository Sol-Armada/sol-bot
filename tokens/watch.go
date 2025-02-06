package tokens

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func Watch(ctx context.Context, out chan TokenRecord) error {
	lastRecordTS := time.Now()

	for {
		select {
		case <-ctx.Done():
			close(out)
			return nil
		default:
		}

		cur, err := tokenStore.Find(ctx, bson.D{
			{Key: "created_at", Value: bson.D{
				{Key: "$gt", Value: lastRecordTS},
			}},
		})
		if err != nil {
			return err
		}

		for cur.Next(ctx) {
			var d TokenRecord
			if err := cur.Decode(&d); err != nil {
				return err
			}

			out <- d
			lastRecordTS = d.CreatedAt
		}

		time.Sleep(1 * time.Second)
	}
}
