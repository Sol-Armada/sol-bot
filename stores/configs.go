package stores

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ConfigsStore struct {
	*store
}

const CONFIGS Collection = "configs"

func newConfigsStore(ctx context.Context, client *mongo.Client, database string) *ConfigsStore {
	_ = client.Database(database).CreateCollection(ctx, string(CONFIGS))
	s := &store{
		Collection: client.Database(database).Collection(string(CONFIGS)),
		ctx:        ctx,
	}
	return &ConfigsStore{s}
}

func (c *Client) GetConfigsStore() (*ConfigsStore, bool) {
	if c.stores == nil {
		return nil, false
	}
	return c.stores.configs, true
}

func (s *ConfigsStore) Get(name string) *mongo.SingleResult {
	filter := bson.D{{Key: "name", Value: name}}
	return s.FindOne(s.ctx, filter)
}

func (s *ConfigsStore) Upsert(name string, config any) error {
	opts := options.FindOneAndReplace().SetUpsert(true)
	if err := s.FindOneAndReplace(s.ctx, bson.D{{Key: "name", Value: name}}, bson.D{{Key: "name", Value: name}, {Key: "value", Value: config}}, opts).Err(); err != nil {
		if err != mongo.ErrNoDocuments {
			return err
		}
	}
	return nil
}
