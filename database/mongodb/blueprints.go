package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

type BlueprintStore struct {
	*mongodb
}

const BLUEPRINT Collection = "blueprint"

func newBlueprintStore(ctx context.Context, client *mongo.Client, database string) *BlueprintStore {
	_ = client.Database(database).CreateCollection(ctx, string(BLUEPRINT))
	s := &mongodb{
		Collection: client.Database(database).Collection(string(BLUEPRINT)),
		ctx:        ctx,
	}
	return &BlueprintStore{s}
}

func (c *Client) GetBlueprintStore() (*BlueprintStore, bool) {
	if c.stores == nil {
		return nil, false
	}
	return c.stores.blueprints, true
}
