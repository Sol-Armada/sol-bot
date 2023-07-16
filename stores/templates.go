package stores

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type transactionsStore struct {
	*mongo.Collection
	ctx context.Context
}

var Templates *templateStore

func (t *templateStore) GetContext() context.Context {
	return t.ctx
}

func (t *templateStore) List(filter interface{}) (*mongo.Cursor, error) {
	return t.Find(t.ctx, filter, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
}

func (t *templateStore) Delete(name string) error {
	if _, err := t.DeleteOne(t.ctx, bson.D{{Key: "name", Value: name}}); err != nil {
		return err
	}
	return nil
}
