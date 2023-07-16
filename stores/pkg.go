package stores

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client
var ctx context.Context

func Setup(cctx context.Context) error {
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

	clientOptions := options.Client().ApplyURI(uri).SetConnectTimeout(5 * time.Second)
	c, err := mongo.Connect(cctx, clientOptions)
	if err != nil {
		return errors.Wrap(err, "creating new store")
	}
	client = c
	ctx = cctx

	Users = &usersStore{
		Collection: client.Database(config.GetStringWithDefault("MONGO.DATABASE", "org")).Collection("users"),
		ctx:        ctx,
	}

	Events = &eventsStore{
		Collection: client.Database(config.GetStringWithDefault("MONGO.DATABASE", "org")).Collection("events"),
		ctx:        ctx,
	}

	Templates = &templateStore{
		Collection: client.Database(config.GetStringWithDefault("MONGO.DATABASE", "org")).Collection("event-templates"),
		ctx:        ctx,
	}

	Transactions = &transactionsStore{
		Collection: client.Database(config.GetStringWithDefault("MONGO.DATABASE", "org")).Collection("transactions"),
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
