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

type store struct {
	*mongo.Collection
	ctx context.Context
}

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

// StoreRegistry provides type-safe access to all stores
type StoreRegistry struct {
	members    *MembersStore
	attendance *AttendanceStore
	configs    *ConfigsStore
	activity   *ActivityStore
	sos        *SOSStore
	tokens     *TokenStore
	raffles    *RaffleStore
	kanban     *KanbanStore
	commands   *CommandsStore
	giveaways  *GiveawaysStore
}

// Store accessor methods
func (s *StoreRegistry) Members() *MembersStore       { return s.members }
func (s *StoreRegistry) Attendance() *AttendanceStore { return s.attendance }
func (s *StoreRegistry) Configs() *ConfigsStore       { return s.configs }
func (s *StoreRegistry) Activity() *ActivityStore     { return s.activity }
func (s *StoreRegistry) SOS() *SOSStore               { return s.sos }
func (s *StoreRegistry) Tokens() *TokenStore          { return s.tokens }
func (s *StoreRegistry) Raffles() *RaffleStore        { return s.raffles }
func (s *StoreRegistry) Kanban() *KanbanStore         { return s.kanban }
func (s *StoreRegistry) Commands() *CommandsStore     { return s.commands }
func (s *StoreRegistry) Giveaways() *GiveawaysStore   { return s.giveaways }

type Client struct {
	*mongo.Client
	stores   *StoreRegistry
	database *mongo.Database
}

// Global client for backward compatibility (will be deprecated)
var client *Client

// NewWithConfig creates a new database client with configuration struct
func NewWithConfig(ctx context.Context, config Config) (*Client, error) {
	return New(ctx, config.Host, config.Port, config.Username, config.Password, config.Database, config.ReplicaSetName)
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

	// Initialize stores
	membersStore := newMembersStore(ctx, mongoClient, database)
	configsStore := newConfigsStore(ctx, mongoClient, database)
	attendanceStore := newAttendanceStore(ctx, mongoClient, database)
	activityStore := newActivityStore(ctx, mongoClient, database)
	sosStore := newSOSStore(ctx, mongoClient, database)
	tokensStore := newTokensStore(ctx, mongoClient, database)
	rafflesStore := newRafflesStore(ctx, mongoClient, database)
	kanbanStore := newKanbanStore(ctx, mongoClient, database)
	commandsStore := newCommandsStore(ctx, mongoClient, database)
	giveawaysStore := newGiveawaysStore(ctx, mongoClient, database)

	storeRegistry := &StoreRegistry{
		members:    membersStore,
		configs:    configsStore,
		attendance: attendanceStore,
		activity:   activityStore,
		sos:        sosStore,
		tokens:     tokensStore,
		raffles:    rafflesStore,
		kanban:     kanbanStore,
		commands:   commandsStore,
		giveaways:  giveawaysStore,
	}

	newClient := &Client{
		Client:   mongoClient,
		stores:   storeRegistry,
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

// Stores returns the store registry for type-safe access
func (c *Client) Stores() *StoreRegistry {
	return c.stores
}

// GetCollection is deprecated, use Stores() instead
func (c *Client) GetCollection(collection Collection) (any, bool) {
	switch collection {
	case MEMBERS:
		return c.stores.members, true
	case CONFIGS:
		return c.stores.configs, true
	case ATTENDANCE:
		return c.stores.attendance, true
	case ACTIVITY:
		return c.stores.activity, true
	case SOS:
		return c.stores.sos, true
	case TOKENS:
		return c.stores.tokens, true
	case RAFFLES:
		return c.stores.raffles, true
	case KANBAN:
		return c.stores.kanban, true
	case COMMANDS:
		return c.stores.commands, true
	case GIVEAWAYS:
		return c.stores.giveaways, true
	default:
		return nil, false
	}
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
