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

const MEMBERS Collection = "members"

func newMembersStore(ctx context.Context, client *mongo.Client, database string) *MembersStore {
	_ = client.Database(database).CreateCollection(ctx, string(MEMBERS))
	s := &store{
		Collection: client.Database(database).Collection(string(MEMBERS)),
		ctx:        ctx,
	}
	return &MembersStore{s}
}

func (c *Client) GetMembersStore() (*MembersStore, bool) {
	storeInterface, ok := c.GetCollection(MEMBERS)
	if !ok {
		return nil, false
	}
	return storeInterface.(*MembersStore), ok
}

func (s *MembersStore) Get(id string) (*mongo.Cursor, error) {
	return s.Aggregate(s.ctx, bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "_id", Value: id}}}},
		// bson.D{{Key: "$lookup", Value: bson.D{
		// 	{Key: "from", Value: "members"},
		// 	{Key: "localField", Value: "recruiter"},
		// 	{Key: "foreignField", Value: "_id"},
		// 	{Key: "as", Value: "recruiter"},
		// }}},
		// bson.D{{Key: "$unwind", Value: bson.D{{Key: "path", Value: "$recruiter"}, {Key: "preserveNullAndEmptyArrays", Value: true}}}},
	})
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

func (s *MembersStore) List(filter any, page, max int) (*mongo.Cursor, error) {
	pipeline := bson.A{
		bson.D{{Key: "$match", Value: filter}},
		// bson.D{{Key: "$lookup", Value: bson.D{
		// 	{Key: "from", Value: "members"},
		// 	{Key: "localField", Value: "recruiter"},
		// 	{Key: "foreignField", Value: "_id"},
		// 	{Key: "as", Value: "recruiter"},
		// }}},
		// bson.D{{Key: "$unwind", Value: bson.D{{Key: "path", Value: "$recruiter"}, {Key: "preserveNullAndEmptyArrays", Value: true}}}},
	}

	if page > 0 {
		pipeline = append(pipeline,
			bson.D{{Key: "$skip", Value: (page - 1) * max}},
			bson.D{{Key: "$limit", Value: max}},
		)
	}

	return s.Aggregate(s.ctx, pipeline)
}

func (s *MembersStore) Upsert(id string, member any) error {
	opts := options.Replace().SetUpsert(true)
	if _, err := s.ReplaceOne(s.ctx, bson.D{{Key: "_id", Value: id}}, member, opts); err != nil {
		return err
	}
	return nil
}

func (s *MembersStore) Delete(id string) error {
	return s.FindOneAndDelete(s.ctx, bson.D{{Key: "_id", Value: id}}).Err()
}
