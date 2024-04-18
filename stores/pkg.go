package stores

import (
	"context"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client
var ctx context.Context

func Setup(cctx context.Context, host string, port int, username string, password string, database string) error {
	log.Debug("creating store")
	usernamePassword := username + ":" + password + "@"
	if usernamePassword == ":@" {
		usernamePassword = ""
	}

	uri := fmt.Sprintf("mongodb://%s%s:%d",
		usernamePassword,
		host,
		port)

	clientOptions := options.Client().ApplyURI(uri).SetConnectTimeout(5 * time.Second)
	c, err := mongo.Connect(cctx, clientOptions)
	if err != nil {
		return errors.Wrap(err, "creating new store")
	}
	client = c
	ctx = cctx

	Users = &usersStore{
		Collection: client.Database(database).Collection("users"),
		ctx:        ctx,
	}

	Events = &eventsStore{
		Collection: client.Database(database).Collection("events"),
		ctx:        ctx,
	}

	Templates = &templateStore{
		Collection: client.Database(database).Collection("event-templates"),
		ctx:        ctx,
	}

	Transactions = &transactionsStore{
		Collection: client.Database(database).Collection("transactions"),
		ctx:        ctx,
	}

	return nil
}

func Disconnect() {
	if err := client.Disconnect(ctx); err != nil {
		log.WithError(err).Error("disconnect from store")
	}
}

func Connected() bool {
	if err := client.Ping(ctx, nil); err != nil {
		return false
	}
	return true
}
