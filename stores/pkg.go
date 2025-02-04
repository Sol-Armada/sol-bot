package stores

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Collection string

const ()

type store struct {
	*mongo.Collection
	ctx context.Context
}

type Client struct {
	*mongo.Client

	databases map[Collection]interface{}

	ctx context.Context
}

var client *Client

func New(ctx context.Context, host string, port int, username string, password string, database string) (*Client, error) {
	usernamePassword := username + ":" + password + "@"
	if usernamePassword == ":@" {
		usernamePassword = ""
	}

	uri := fmt.Sprintf("mongodb://%s%s:%d",
		usernamePassword,
		host,
		port)

	clientOptions := options.Client().ApplyURI(uri).SetConnectTimeout(5 * time.Second)
	mongoClient, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, errors.Wrap(err, "creating new store")
	}

	client = &Client{
		Client:    mongoClient,
		databases: map[Collection]interface{}{},
		ctx:       ctx,
	}

	if ok := client.Connected(); !ok {
		return nil, errors.New("unable to connect to store")
	}

	client.databases[MEMBERS] = newMembersStore(ctx, client.Client, database)
	client.databases[CONFIGS] = newConfigsStore(ctx, client.Client, database)
	client.databases[ATTENDANCE] = newAttendanceStore(ctx, client.Client, database)
	client.databases[ACTIVITY] = newActivityStore(ctx, client.Client, database)
	client.databases[SOS] = newSOSStore(ctx, client.Client, database)
	client.databases[TOKENS] = newTokensStore(ctx, client.Client, database)

	return client, nil
}

func Get() *Client {
	return client
}

func (c *Client) GetCollection(collection Collection) (interface{}, bool) {
	if c.databases[collection] == nil {
		return nil, false
	}
	return c.databases[collection], true
}

func (c *Client) Disconnect() {
	_ = c.Client.Disconnect(c.ctx)
}

func (c *Client) Connected() bool {
	if err := c.Ping(c.ctx, nil); err != nil {
		return false
	}
	return true
}
