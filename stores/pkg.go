package stores

import (
	"context"
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Store struct {
	users  *mongo.Collection
	client *mongo.Client
	ctx    context.Context
}

var Storage *Store

func New(ctx context.Context) (*Store, error) {
	log.Debug("creating store")
	password := strings.ReplaceAll(config.GetString("MONGO.PASSWORD"), "@", `%40`)
	usernamePassword := config.GetString("MONGO.USERNAME") + ":" + password + "@"
	if usernamePassword == ":@" {
		usernamePassword = ""
	}

	uri := fmt.Sprintf("mongodb://%s%s:%d",
		usernamePassword,
		config.GetStringWithDefault("mongo.host", "localhost"),
		config.GetIntWithDefault("mongo.port", 27017))
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, errors.Wrap(err, "creating new store")
	}

	collection := client.Database(config.GetStringWithDefault("MONGO.DATABASE", "org")).Collection("users")

	Storage = &Store{
		client: client,
		users:  collection,
		ctx:    ctx,
	}

	return Storage, nil
}

func (s *Store) Disconnect() {
	if err := s.client.Disconnect(s.ctx); err != nil {
		log.WithError(err).Error("disconnect from store")
	}
}

func (s *Store) SaveUser(id string, u interface{}) error {
	log.WithField("id", id).Debug("saving user to mongo")
	opts := options.Replace().SetUpsert(true)
	if _, err := s.users.ReplaceOne(s.ctx, bson.D{{Key: "_id", Value: id}}, u, opts); err != nil {
		return errors.Wrap(err, "saving user to mongo")
	}
	return nil
}

func (s *Store) SaveUsers(u map[string]interface{}) error {
	log.WithField("count", len(u)).Info("saving users to mongo")
	for id, user := range u {
		if err := s.SaveUser(id, user); err != nil {
			return errors.Wrap(err, "saving users to mongo")
		}
	}

	return nil
}

func (s *Store) GetUser(id string) *mongo.SingleResult {
	filter := bson.D{{Key: "_id", Value: id}}
	return s.users.FindOne(s.ctx, filter)
}

func (s *Store) GetUsers() (*mongo.Cursor, error) {
	return s.users.Find(s.ctx, bson.M{})
}

func (s *Store) DeleteUser(id string) error {
	filter := bson.D{{Key: "_id", Value: id}}
	if _, err := s.users.DeleteOne(s.ctx, filter); err != nil {
		return errors.Wrap(err, "deleting a user from mongo")
	}
	return nil
}
