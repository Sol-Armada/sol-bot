package members

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/bson"
)

func Watch(ctx context.Context, out chan Member) error {
	stream, err := membersStore.Watch(ctx, bson.D{})
	if err != nil {
		return err
	}
	defer stream.Close(ctx)

	for stream.Next(ctx) {
		var event bson.M
		if err := stream.Decode(&event); err != nil {
			return err
		}

		operationType, ok := event["operationType"].(string)
		if !ok {
			slog.Error("operationType not found in event", "event", event)
			continue
		}

		if operationType != "insert" {
			continue
		}

		memberRaw, ok := event["fullDocument"].(bson.M)
		if !ok {
			slog.Error("fullDocument not found in event", "event", event)
			continue
		}

		bytes, err := bson.Marshal(memberRaw)
		if err != nil {
			slog.Error("failed to marshal member", "err", err)
			continue
		}

		var member Member
		if err := bson.Unmarshal(bytes, &member); err != nil {
			slog.Error("failed to unmarshal member", "err", err)
			continue
		}

		out <- member
	}
	return ctx.Err()
}
