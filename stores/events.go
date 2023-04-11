package stores

import (
	"time"

	"github.com/pkg/errors"
	apierrors "github.com/sol-armada/admin/errors"
	"github.com/sol-armada/admin/events/status"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Store) GetEvents(filter interface{}) (*mongo.Cursor, error) {
	return s.events.Find(s.ctx, filter, options.Find().SetSort(bson.D{{Key: "start", Value: 1}}))
}

func (s *Store) GetNextEvent() *mongo.SingleResult {
	sr := s.events.FindOne(s.ctx, bson.D{
		{
			Key: "$and",
			Value: bson.A{
				bson.D{{Key: "status", Value: bson.D{{Key: "$lt", Value: status.Live}}}},
				bson.D{{Key: "end", Value: bson.D{{Key: "$gt", Value: time.Now()}}}},
			},
		},
	}, options.FindOne().SetSort(bson.D{{Key: "start", Value: 1}}))
	return sr
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

	if start, ok := e["start"].(int64); ok {
		t := time.UnixMilli(start)
		e["start"] = primitive.NewDateTimeFromTime(t)
	}

	if end, ok := e["end"].(int64); ok {
		t := time.UnixMilli(end)
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
