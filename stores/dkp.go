package stores

import (
	"context"

	"github.com/sol-armada/sol-bot/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DKPStore struct {
	*store
}

const DKP Collection = "dkp"

func newDKPStore(ctx context.Context, client *mongo.Client, database string) *DKPStore {
	_ = client.Database(database).CreateCollection(ctx, string(DKP), &options.CreateCollectionOptions{
		TimeSeriesOptions: &options.TimeSeriesOptions{
			TimeField: "created_at",
			MetaField: utils.StringPointer("member_id"),
		},
	})
	s := &store{
		Collection: client.Database(database).Collection(string(DKP)),
		ctx:        ctx,
	}
	return &DKPStore{s}
}

func (c *Client) GetDKPStore() (*DKPStore, bool) {
	storeInterface, ok := c.GetCollection(DKP)
	if !ok {
		return nil, false
	}
	return storeInterface.(*DKPStore), ok
}

func (s *DKPStore) Insert(dkp any) error {
	_, err := s.InsertOne(s.ctx, dkp)
	return err
}

// Get all dkp grouping by member id
func (s *DKPStore) GetAll() (*mongo.Cursor, error) {
	aggregate := []bson.M{
		{
			"$group": bson.M{
				"_id": "$member_id",
				"dkp": bson.M{
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

func (s *DKPStore) Get(id string) (*mongo.SingleResult, error) {
	res := s.FindOne(s.ctx, bson.D{{Key: "_id", Value: id}})
	err := res.Err()
	return res, err
}

func (s *DKPStore) GetAllBalances() (*mongo.Cursor, error) {
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
