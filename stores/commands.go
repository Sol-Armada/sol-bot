package stores

import (
	"context"

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
