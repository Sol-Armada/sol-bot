package stores

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CommandsStore struct {
	*store
}

const COMMANDS Collection = "commands"

func newCommandsStore(ctx context.Context, client *mongo.Client, database string) *CommandsStore {
	_ = client.Database(database).CreateCollection(ctx, string(COMMANDS), &options.CreateCollectionOptions{
		TimeSeriesOptions: &options.TimeSeriesOptions{
			TimeField: "when",
			MetaField: new("meta"),
		},
	})
	s := &store{
		Collection: client.Database(database).Collection(string(COMMANDS)),
		ctx:        ctx,
	}
	return &CommandsStore{s}
}

func (c *Client) GetCommandsStore() (*CommandsStore, bool) {
	if c.stores == nil {
		return nil, false
	}
	return c.stores.commands, true
}

func (s *CommandsStore) Create(command any) error {
	_, err := s.InsertOne(s.ctx, command)
	return err
}

func (s *CommandsStore) CountsByName() (map[string]int64, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$name"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}
	cursor, err := s.Aggregate(s.ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(s.ctx)

	counts := make(map[string]int64)
	for cursor.Next(s.ctx) {
		var result struct {
			Name  string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		counts[result.Name] = result.Count
	}
	return counts, nil
}
