package stores

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type eventsStore struct {
	*mongo.Collection
	ctx context.Context
}

var Events *eventsStore

func (e *eventsStore) GetContext() context.Context {
	return e.ctx
}

func (e *eventsStore) List(filter interface{}) (*mongo.Cursor, error) {
	return e.Find(e.ctx, filter, options.Find().SetSort(bson.D{{Key: "start", Value: 1}}))
}

func (e *eventsStore) Get(id string) *mongo.SingleResult {
	return e.FindOne(e.ctx, bson.D{{Key: "_id", Value: id}})
}
