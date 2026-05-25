package stores

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RaffleStore struct {
	*store
}

const RAFFLES Collection = "raffles"

func newRafflesStore(ctx context.Context, client *mongo.Client, database string) *RaffleStore {
	_ = client.Database(database).CreateCollection(ctx, string(RAFFLES))
	s := &store{
		Collection: client.Database(database).Collection(string(RAFFLES)),
		ctx:        ctx,
	}
	return &RaffleStore{s}
}

func (c *Client) GetRafflesStore() (*RaffleStore, bool) {
	if c.stores == nil {
		return nil, false
	}
	return c.stores.raffles, true
}

func (s *RaffleStore) Get(id string) *mongo.SingleResult {
	filter := bson.D{{Key: "_id", Value: id}}
	return s.FindOne(s.ctx, filter)
}

func (s *RaffleStore) GetLatest() (*mongo.Cursor, error) {
	opts := options.Find().SetSort(bson.D{{Key: "createdat", Value: -1}}).SetLimit(1)
	return s.Find(s.ctx, bson.D{}, opts)
}

func (s *RaffleStore) Upsert(id string, raffle any) error {
	opts := options.FindOneAndReplace().SetUpsert(true)
	if err := s.FindOneAndReplace(s.ctx, bson.D{{Key: "_id", Value: id}}, raffle, opts).Err(); err != nil {
		if err != mongo.ErrNoDocuments {
			return err
		}
	}
	return nil
}

func (s *RaffleStore) Delete(id string) error {
	filter := bson.D{{Key: "_id", Value: id}}
	_, err := s.DeleteOne(s.ctx, filter)
	return err
}
