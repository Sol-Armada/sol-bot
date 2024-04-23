package stores

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ConfigsStore struct {
	*store
}

func newConfigsStore(ctx context.Context, client *mongo.Client, database string) *ConfigsStore {
	s := &store{
		Collection: client.Database(database).Collection("configs"),
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

func (s *ConfigsStore) Update(name string, config any) error {
	if err := s.FindOneAndReplace(s.ctx, bson.D{{Key: "name", Value: name}}, config).Err(); err != nil {
		if err != mongo.ErrNoDocuments {
			return err
		}
		_, err = s.InsertOne(s.ctx, config)
		return err
	}
	return nil
}
