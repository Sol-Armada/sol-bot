package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Collection string

// Config holds the database connection configuration
type Config struct {
	Host           string
	Port           int
	Username       string
	Password       string
	Database       string
	ReplicaSetName string
	ConnectTimeout time.Duration
}

type Client struct {
	*mongo.Client
	database *mongo.Database
}

// Global client for backward compatibility (will be deprecated)
var client *Client

// NewWithConfig creates a new database client with configuration struct
func NewWithConfig(ctx context.Context, config Config) (*Client, error) {
	connectCtx := ctx
	if config.ConnectTimeout > 0 {
		var cancel context.CancelFunc
		connectCtx, cancel = context.WithTimeout(ctx, config.ConnectTimeout)
		defer cancel()
	}

	return New(connectCtx, config.Host, config.Port, config.Username, config.Password, config.Database, config.ReplicaSetName)
}

func New(ctx context.Context, host string, port int, username, password, database, replicaSetName string) (*Client, error) {
	usernamePassword := username + ":" + password + "@"
	if usernamePassword == ":@" {
		usernamePassword = ""
	}

	if replicaSetName != "" {
		replicaSetName = fmt.Sprintf("/?replicaSet=%s", replicaSetName)
	}

	uri := fmt.Sprintf("mongodb://%s%s:%d%s",
		usernamePassword,
		host,
		port,
		replicaSetName)

	clientOptions := options.Client().ApplyURI(uri).SetConnectTimeout(5 * time.Second)
	mongoClient, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, errors.Wrap(err, "creating new store")
	}

	db := mongoClient.Database(database)

	newClient := &Client{
		Client:   mongoClient,
		database: db,
	}

	if ok := newClient.Connected(ctx); !ok {
		return nil, errors.New("unable to connect to store")
	}

	// Set global client for backward compatibility
	client = newClient

	return newClient, nil
}

func Get() *Client {
	return client
}

func (c *Client) Disconnect(ctx context.Context) error {
	return c.Client.Disconnect(ctx)
}

func (c *Client) Connected(ctx context.Context) bool {
	if err := c.Ping(ctx, nil); err != nil {
		return false
	}
	return true
}
