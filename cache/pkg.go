package cache

import (
	"context"
	"encoding/json"

	"github.com/apex/log"
	"github.com/redis/go-redis/v9"
	"github.com/sol-armada/sol-bot/config"
)

type cache struct {
	redis *redis.Client

	ctx context.Context
}

var Cache *cache

func Setup() {
	url := config.GetStringWithDefault("REDIS.URL", "localhost:6379")
	Cache = &cache{
		redis: redis.NewClient(
			&redis.Options{
				Addr:     url,
				DB:       config.GetIntWithDefault("REDIS.DB", 0),
				Password: config.GetStringWithDefault("REDIS.PASSWORD", ""),
			}),
		ctx: context.Background(),
	}
}

func (c *cache) Set(key string, value interface{}) {
	c.redis.Set(c.ctx, key, value, 0)
}

func (c *cache) Get(key string) interface{} {
	cmd := c.redis.Get(c.ctx, key)
	return cmd.Val()
}

func (c *cache) Scan(key string) []string {
	keys := []string{}
	iter := c.redis.Scan(c.ctx, 0, key, 0).Iterator()
	for iter.Next(c.ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		log.WithError(err).Error("scanning keys")
	}
	return keys
}

func (c *cache) Delete(key string) {
	c.redis.Del(c.ctx, key)
}

func (c *cache) GetUser(id string) map[string]interface{} {
	rawUser := c.Get("user:" + id).(string)
	if rawUser == "" {
		return nil
	}
	user := map[string]interface{}{}
	if err := json.Unmarshal([]byte(rawUser), &user); err != nil {
		log.WithField("user", id).WithError(err).Error("unmarshalling user")
	}
	return user
}

func (c *cache) SetUser(id string, user map[string]interface{}) {
	rawUser, _ := json.Marshal(user)
	c.Set("user:"+id, string(rawUser))
}

func (c *cache) DeleteUser(id string) {
	c.Delete("user:" + id)
}

func (c *cache) GetUsers() []map[string]interface{} {
	keys := c.Scan("user:*")
	users := []map[string]interface{}{}
	for _, key := range keys {
		users = append(users, c.GetUser(key[5:]))
	}
	return users
}
