package stores

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TokenStore struct {
	*store
}

const TOKENS Collection = "tokens"

func newTokensStore(ctx context.Context, client *mongo.Client, database string) *TokenStore {
	_ = client.Database(database).CreateCollection(ctx, string(TOKENS), &options.CreateCollectionOptions{
		TimeSeriesOptions: &options.TimeSeriesOptions{
			TimeField: "created_at",
			MetaField: new("member_id"),
		},
	})
	s := &store{
		Collection: client.Database(database).Collection(string(TOKENS)),
		ctx:        ctx,
	}
	return &TokenStore{s}
}

func (c *Client) GetTokensStore() (*TokenStore, bool) {
	if c.stores == nil {
		return nil, false
	}
	return c.stores.tokens, true
}

func (s *TokenStore) Insert(tokenRecord any) error {
	_, err := s.InsertOne(s.ctx, tokenRecord)
	return err
}

// Get all token records grouping by member id
func (s *TokenStore) GetAllGrouped() (*mongo.Cursor, error) {
	aggregate := []bson.M{
		{
			"$group": bson.M{
				"_id": "$member_id",
				"token_records": bson.M{
					"$push": "$$ROOT",
				},
			},
		},
	}

	cursor, err := s.Aggregate(s.ctx, aggregate)
	if err != nil {
		return nil, err
	}

	return cursor, nil
}

func (s *TokenStore) GetAll() (*mongo.Cursor, error) {
	cursor, err := s.Find(s.ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	return cursor, nil
}

func (s *TokenStore) Get(id string) (*mongo.SingleResult, error) {
	res := s.FindOne(s.ctx, bson.D{{Key: "_id", Value: id}})
	err := res.Err()
	return res, err
}

func (s *TokenStore) GetAllBalances() (*mongo.Cursor, error) {
	aggregate := bson.A{
		bson.D{
			{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$member_id"},
				{Key: "balance", Value: bson.D{
					{Key: "$sum", Value: "$amount"},
				}},
			}},
		},
		bson.D{
			{Key: "$addFields", Value: bson.D{
				{Key: "balance", Value: bson.D{
					{Key: "$sum", Value: "$balance"},
				}},
			}},
		},
	}

	cursor, err := s.Aggregate(s.ctx, aggregate)
	if err != nil {
		return nil, err
	}
	return cursor, nil
}

func (s *TokenStore) GetByMemberIdAndAttendanceId(memberId, attendanceId string) (*mongo.Cursor, error) {
	filter := bson.D{
		{Key: "member_id", Value: memberId},
		{Key: "attendance_id", Value: attendanceId},
	}

	cursor, err := s.Find(s.ctx, filter)
	if err != nil {
		return nil, err
	}
	return cursor, nil
}
