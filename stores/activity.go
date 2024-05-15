package stores

import (
	"context"

	"github.com/sol-armada/sol-bot/utils"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ActivityStore struct {
	*store
}

func newActivityStore(ctx context.Context, client *mongo.Client, database string) *ActivityStore {
	var col *mongo.Collection
	if err := client.Database(database).CreateCollection(ctx, string(ACTIVITY), &options.CreateCollectionOptions{
		TimeSeriesOptions: &options.TimeSeriesOptions{
			TimeField: "when",
			MetaField: utils.StringPointer("meta"),
		},
	}); err != nil {
		col = client.Database(database).Collection(string(ACTIVITY))
	}

	s := &store{
		Collection: col,
		ctx:        ctx,
	}
	return &ActivityStore{s}
}

func (s *ActivityStore) Create(activity any) error {
	_, err := s.InsertOne(s.ctx, activity)
	return err
}
