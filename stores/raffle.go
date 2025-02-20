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
	storeInterface, ok := c.GetCollection(RAFFLES)
	if !ok {
		return nil, false
	}
	return storeInterface.(*RaffleStore), ok
}

func (s *RaffleStore) Get(id string) *mongo.SingleResult {
	filter := bson.D{{Key: "_id", Value: id}}
	return s.FindOne(s.ctx, filter)
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
