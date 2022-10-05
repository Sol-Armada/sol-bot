package users

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/apex/log"
	rds "github.com/go-redis/redis/v9"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/config"
)

type Store struct {
	ctx  context.Context
	sess *rds.Client
}

var storage *Store

func NewStorage(ctx context.Context) {
	storage = &Store{
		ctx,
		rds.NewClient(&rds.Options{
			Addr:     config.GetStringWithDefault("REDIS.ADDRESS", "localhost:6379"),
			Password: config.GetStringWithDefault("REDIS.PASSWORD", ""),
			DB:       config.GetIntWithDefault("REDIS.DB", 0),
		}),
	}
}

func GetStorage() *Store {
	if storage == nil {
		NewStorage(context.Background())
	}
	return storage
}

func (s *Store) GetAdmins() (map[string]*Admin, error) {
	admins := map[string]*Admin{}

	val, err := s.sess.Keys(s.ctx, "admin:*").Result()
	if err != nil {
		return nil, err
	}

	for _, key := range val {
		v, err := s.sess.Get(s.ctx, key).Result()
		if err != nil {
			return nil, err
		}

		var a *Admin
		if err := json.Unmarshal([]byte(v), &a); err != nil {
			return nil, err
		}

		admins[a.User.Id] = a
	}

	return admins, nil
}

func (s *Store) SaveAdmin(a *Admin) error {
	adminJson, err := json.Marshal(a)
	if err != nil {
		return err
	}
	if err != nil {
		log.WithError(err).Error("saving admin to redis")
	}
	s.sess.Set(s.ctx, fmt.Sprintf("admin:%s", a.User.Id), string(adminJson), 0)
	return nil
}

func (s *Store) SaveUser(u *User) error {
	userJson, err := json.Marshal(u)
	if err != nil {
		return err
	}
	if err != nil {
		return errors.Wrap(err, "saving user to redis")
	}
	s.sess.Set(s.ctx, fmt.Sprintf("user:%s", u.Id), string(userJson), 0)
	return nil
}

func (s *Store) SaveUsers(u []*User) error {
	for _, user := range u {
		if err := s.SaveUser(user); err != nil {
			return errors.Wrap(err, "saving users")
		}
	}

	return nil
}

func (s *Store) GetUser(id string) (*User, error) {
	val, err := s.sess.Get(s.ctx, fmt.Sprintf("user:%s", id)).Result()
	if err != nil {
		return nil, err
	}

	var u *User
	if err := json.Unmarshal([]byte(val), &u); err != nil {
		return nil, err
	}

	return u, nil
}

func (s *Store) GetUsers() ([]*User, error) {
	us := []*User{}

	val, err := s.sess.Keys(s.ctx, "user:*").Result()
	if err != nil {
		return nil, err
	}

	for _, key := range val {
		v, err := s.sess.Get(s.ctx, key).Result()
		if err != nil {
			return nil, err
		}

		var u *User
		if err := json.Unmarshal([]byte(v), &u); err != nil {
			return nil, err
		}

		us = append(us, u)
	}

	return us, nil
}
