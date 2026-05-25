package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ActivityStore struct {
	*mongodb
}

const ACTIVITY Collection = "activity"

func newActivityStore(ctx context.Context, client *mongo.Client, database string) *ActivityStore {
	_ = client.Database(database).CreateCollection(ctx, string(ACTIVITY), &options.CreateCollectionOptions{
		TimeSeriesOptions: &options.TimeSeriesOptions{
			TimeField: "when",
			MetaField: new("meta"),
		},
	})
	s := &mongodb{
		Collection: client.Database(database).Collection(string(ACTIVITY)),
		ctx:        ctx,
	}
	return &ActivityStore{s}
}

func (c *Client) GetActivityStore() (*ActivityStore, bool) {
	if c.stores == nil {
		return nil, false
	}
	return c.stores.activity, true
}

func (s *ActivityStore) Create(activity any) error {
	_, err := s.InsertOne(s.ctx, activity)
	return err
}
