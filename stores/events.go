package stores

import (
	"time"

	"github.com/pkg/errors"
	apierrors "github.com/sol-armada/admin/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Store) GetEvents(filter interface{}) (*mongo.Cursor, error) {
	return s.events.Find(s.ctx, filter)
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

	if start, ok := e["start"].(string); ok {
		t, err := time.Parse(time.RFC3339, start)
		if err != nil {
			return err
		}
		e["start"] = primitive.NewDateTimeFromTime(t)
	}

	if end, ok := e["end"].(string); ok {
		t, err := time.Parse(time.RFC3339, end)
		if err != nil {
			return err
		}
		e["end"] = primitive.NewDateTimeFromTime(t)
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
