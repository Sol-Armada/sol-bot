package stores

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SOSStore struct {
	*store
}

const SOS Collection = "sos"

func newSOSStore(ctx context.Context, client *mongo.Client, database string) *MembersStore {
	_ = client.Database(database).CreateCollection(ctx, string(SOS))
	s := &store{
		Collection: client.Database(database).Collection(string(SOS)),
		ctx:        ctx,
	}
	return &MembersStore{s}
}

func (s *SOSStore) Upsert(id string, ticket any) error {
	opts := options.Replace().SetUpsert(true)
	if _, err := s.ReplaceOne(s.ctx, bson.D{{Key: "_id", Value: id}}, ticket, opts); err != nil {
		return err
	}
	return nil
}

func (s *SOSStore) GetSOSTicketsByMemberId(memberId string) (*mongo.Cursor, error) {
	return s.Aggregate(s.ctx, bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "member_id", Value: memberId}}}},
	})
}

func (s *SOSStore) GetSOSTickets() (*mongo.Cursor, error) {
	return s.Find(s.ctx, bson.D{})
}

func (c *Client) GetSOSStore() (*SOSStore, bool) {
	storeInterface, ok := c.GetCollection(SOS)
	if !ok {
		return nil, false
	}
	return storeInterface.(*SOSStore), ok
}
