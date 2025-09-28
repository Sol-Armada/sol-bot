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
	// Note: CreateCollection ignores "collection already exists" errors
	_ = client.Database(database).CreateCollection(ctx, string(MEMBERS))
	s := &store{
		Collection: client.Database(database).Collection(string(MEMBERS)),
		ctx:        ctx,
	}
	return &MembersStore{s}
}

func (c *Client) GetMembersStore() (*MembersStore, bool) {
	if c.stores == nil {
		return nil, false
	}
	return c.stores.members, true
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

func (s *MembersStore) BulkUpsert(members []interface{}) error {
	if len(members) == 0 {
		return nil
	}

	var operations []mongo.WriteModel

	for _, member := range members {
		memberMap, ok := member.(map[string]interface{})
		if !ok {
			continue
		}

		id := memberMap["_id"]

		operation := mongo.NewReplaceOneModel().
			SetFilter(bson.D{{Key: "_id", Value: id}}).
			SetReplacement(memberMap).
			SetUpsert(true)

		operations = append(operations, operation)
	}

	if len(operations) == 0 {
		return nil
	}

	_, err := s.BulkWrite(s.ctx, operations)
	return err
}

func (s *MembersStore) GetIDsOnly() ([]string, error) {
	cursor, err := s.Find(s.ctx, bson.D{}, options.Find().SetProjection(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(s.ctx)

	var ids []string
	for cursor.Next(s.ctx) {
		var result struct {
			ID string `bson:"_id"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		ids = append(ids, result.ID)
	}

	return ids, cursor.Err()
}
