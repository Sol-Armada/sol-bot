package stores

import (
	"context"
	"strings"

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
	filter := bson.D{{Key: "name", Value: strings.ToLower(name)}}
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

// UpsertOverride updates or inserts a config override with full metadata
func (s *ConfigsStore) UpsertOverride(override any) error {
	opts := options.Update().SetUpsert(true)
	
	overrideMap, ok := override.(bson.M)
	if !ok {
		// Convert to bson.M if needed
		data, err := bson.Marshal(override)
		if err != nil {
			return err
		}
		if err := bson.Unmarshal(data, &overrideMap); err != nil {
			return err
		}
	}
	
	name, ok := overrideMap["name"]
	if !ok {
		return mongo.ErrNoDocuments
	}
	
	_, err := s.UpdateOne(s.ctx, bson.M{"name": name}, bson.M{"$set": overrideMap}, opts)
	return err
}

// GetAll returns all config documents
func (s *ConfigsStore) GetAll() (*mongo.Cursor, error) {
	return s.Find(s.ctx, bson.M{})
}
