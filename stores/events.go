package stores

import (
	"github.com/pkg/errors"
	apierrors "github.com/sol-armada/admin/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Store) GetEvents() (*mongo.Cursor, error) {
	return s.events.Find(s.ctx, bson.M{})
}

func (s *Store) GetEvent(id string) (map[string]interface{}, error) {
	event := map[string]interface{}{}
	if err := s.events.FindOne(s.ctx, bson.D{{Key: "_id", Value: id}}).Decode(&event); err != nil {
		return nil, err
	}
	return event, nil
}

func (s *Store) SaveEvent(e map[string]interface{}) error {
	id, ok := e["_id"].(string)
	if !ok {
		return apierrors.ErrMissingId
	}

	opts := options.Replace().SetUpsert(true)
	if _, err := s.events.ReplaceOne(s.ctx, bson.D{{Key: "_id", Value: id}}, e, opts); err != nil {
		return errors.Wrap(err, "saving event")
	}

	return nil
}

func (s *Store) DeleteEvent(e map[string]interface{}) error {
	id, ok := e["_id"].(string)
	if !ok {
		return apierrors.ErrMissingId
	}

	opts := options.Delete()
	if _, err := s.events.DeleteOne(s.ctx, bson.D{{Key: "_id", Value: id}}, opts); err != nil {
		return errors.Wrap(err, "deleting event")
	}

	return nil
}
