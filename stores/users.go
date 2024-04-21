package stores

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type usersStore struct {
	*mongo.Collection
	ctx context.Context
}

var Users *usersStore

func (u *usersStore) GetContext() context.Context {
	return u.ctx
}

func (u *usersStore) Get(id string) *mongo.SingleResult {
	filter := bson.D{{Key: "_id", Value: id}}
	return u.FindOne(u.ctx, filter)
}

func (u *usersStore) List(filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	return u.Find(u.ctx, filter, opts...)
}

func (u *usersStore) Update(id string, user any) error {
	if err := u.FindOneAndReplace(u.ctx, bson.D{{Key: "_id", Value: id}}, user).Err(); err != nil {
		if err != mongo.ErrNoDocuments {
			return err
		}
		_, err = u.InsertOne(u.ctx, user)
		return err
	}
	return nil
}
