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

var Transactions *transactionsStore

func (t *transactionsStore) GetContext() context.Context {
	return t.ctx
}

func (t *transactionsStore) List(filter interface{}) (*mongo.Cursor, error) {
	return t.Find(t.ctx, filter, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
}
