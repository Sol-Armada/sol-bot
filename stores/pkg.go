package stores

import (
	"context"
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Store struct {
	users        *mongo.Collection
	events       *mongo.Collection
	transactions *mongo.Collection
	client       *mongo.Client
	ctx          context.Context
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

	usersCollection := client.Database(config.GetStringWithDefault("MONGO.DATABASE", "org")).Collection("users")
	eventsCollection := client.Database(config.GetStringWithDefault("MONGO.DATABASE", "org")).Collection("events")
	transactionsCollection := client.Database(config.GetStringWithDefault("MONGO.DATABASE", "org")).Collection("transactions")

	Storage = &Store{
		client:       client,
		users:        usersCollection,
		events:       eventsCollection,
		transactions: transactionsCollection,
		ctx:          ctx,
	}

	return Storage, nil
}

func (s *Store) Disconnect() {
	if err := s.client.Disconnect(s.ctx); err != nil {
		log.WithError(err).Error("disconnect from store")
	}
}
