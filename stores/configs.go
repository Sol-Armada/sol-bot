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

func newConfigsStore(ctx context.Context, client *mongo.Client, database string) *ConfigsStore {
	var col *mongo.Collection
	if err := client.Database(database).CreateCollection(ctx, string(CONFIGS)); err != nil {
		col = client.Database(database).Collection(string(CONFIGS))
	}
	s := &store{
		Collection: col,
		ctx:        ctx,
	}
	return &ConfigsStore{s}
}

func (s *ConfigsStore) Create(config any) error {
	_, err := s.InsertOne(s.ctx, config)
	return err
}

func (s *ConfigsStore) Get(name string) *mongo.SingleResult {
	filter := bson.D{{Key: "name", Value: name}}
	return s.FindOne(s.ctx, filter)
}

func (s *ConfigsStore) Upsert(name string, config any) error {
	opts := options.FindOneAndReplace().SetUpsert(true)
	if err := s.FindOneAndReplace(s.ctx, bson.D{{Key: "name", Value: name}}, config, opts).Err(); err != nil {
		return err
	}
	return nil
}
