package stores

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MembersStore struct {
	*store
}

func newMembersStore(ctx context.Context, client *mongo.Client, database string) *MembersStore {
	s := &store{
		Collection: client.Database(database).Collection("members"),
		ctx:        ctx,
	}

	return &MembersStore{s}
}

func (s *MembersStore) Get(id string) *mongo.SingleResult {
	filter := bson.D{{Key: "_id", Value: id}}
	return s.FindOne(s.ctx, filter)
}

func (s *MembersStore) GetRandom(max int, maxRank int) ([]map[string]interface{}, error) {
	cur, err := s.Aggregate(s.ctx, bson.A{
		bson.D{
			{Key: "$match",
				Value: bson.D{
					{Key: "rank",
						Value: bson.D{
							{Key: "$lte", Value: maxRank},
							{Key: "$ne", Value: 0},
						},
					},
				},
			},
		},
		bson.D{{Key: "$sample", Value: bson.D{{Key: "size", Value: max}}}},
	})
	if err != nil {
		return nil, err
	}

	members := []map[string]interface{}{}
	for cur.Next(s.ctx) {
		member := map[string]interface{}{}
		if err := cur.Decode(&member); err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return members, nil
}

func (s *MembersStore) List(filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	return s.Find(s.ctx, filter, opts...)
}

func (s *MembersStore) Upsert(id string, member any) error {
	opts := options.FindOneAndReplace().SetUpsert(true)
	if err := s.FindOneAndReplace(s.ctx, bson.D{{Key: "_id", Value: id}}, member, opts).Err(); err != nil {
		return err
	}
	return nil
}

func (s *MembersStore) Delete(id string) error {
	return s.FindOneAndDelete(s.ctx, bson.D{{Key: "_id", Value: id}}).Err()
}
