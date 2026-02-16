package stores

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type GiveawaysStore struct {
	*store
}

const GIVEAWAYS Collection = "Giveaways"

func newGiveawaysStore(ctx context.Context, client *mongo.Client, database string) *GiveawaysStore {
	_ = client.Database(database).CreateCollection(ctx, string(GIVEAWAYS))
	s := &store{
		Collection: client.Database(database).Collection(string(GIVEAWAYS)),
		ctx:        ctx,
	}
	return &GiveawaysStore{s}
}

func (c *Client) GetGiveawaysStore() (*GiveawaysStore, bool) {
	if c.stores == nil {
		return nil, false
	}
	return c.stores.giveaways, true
}

func (g *GiveawaysStore) GetAll() (*mongo.Cursor, error) {
	return g.Find(g.ctx, bson.D{
		{Key: "ended", Value: false},
	})
}

func (g *GiveawaysStore) UpsertAll(giveaways map[string]any) error {
	var models []mongo.WriteModel
	for id, giveaway := range giveaways {
		models = append(models, mongo.NewReplaceOneModel().SetFilter(bson.D{{Key: "_id", Value: id}}).SetReplacement(giveaway).SetUpsert(true))
	}

	_, err := g.BulkWrite(g.ctx, models)
	if err != nil {
		return errors.Join(err, errors.New("failed to upsert giveaways to store"))
	}

	return nil
}
