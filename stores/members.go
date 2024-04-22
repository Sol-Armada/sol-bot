package stores

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type membersStore struct {
	*mongo.Collection
	ctx context.Context
}

var Members *membersStore

func (m *membersStore) GetContext() context.Context {
	return m.ctx
}

func (m *membersStore) Get(id string) *mongo.SingleResult {
	filter := bson.D{{Key: "_id", Value: id}}
	return m.FindOne(m.ctx, filter)
}

func (m *membersStore) List(filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	return m.Find(m.ctx, filter, opts...)
}

func (m *membersStore) Update(id string, member any) error {
	if err := m.FindOneAndReplace(m.ctx, bson.D{{Key: "_id", Value: id}}, member).Err(); err != nil {
		if err != mongo.ErrNoDocuments {
			return err
		}
		_, err = m.InsertOne(m.ctx, member)
		return err
	}
	return nil
}

func (m *membersStore) Delete(id string) error {
	return m.FindOneAndDelete(m.ctx, bson.D{{Key: "_id", Value: id}}).Err()
}
