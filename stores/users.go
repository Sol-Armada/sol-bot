package stores

import (
	"github.com/apex/log"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Store) SaveUser(id string, u interface{}) error {
	log.WithField("id", id).Debug("saving user to mongo")
	opts := options.Replace().SetUpsert(true)
	if _, err := s.users.ReplaceOne(s.ctx, bson.D{{Key: "_id", Value: id}}, u, opts); err != nil {
		return errors.Wrap(err, "saving user to mongo")
	}
	return nil
}

func (s *Store) SaveUsers(u map[string]interface{}) error {
	log.WithField("count", len(u)).Info("saving users to mongo")
	for id, user := range u {
		if err := s.SaveUser(id, user); err != nil {
			return errors.Wrap(err, "saving users to mongo")
		}
	}

	return nil
}

func (s *Store) GetUser(id string) *mongo.SingleResult {
	filter := bson.D{{Key: "_id", Value: id}}
	return s.users.FindOne(s.ctx, filter)
}

func (s *Store) GetUsers(filter interface{}) (*mongo.Cursor, error) {
	return s.users.Find(s.ctx, filter)
}

func (s *Store) GetRandomUsers(max int, maxRank int) (*mongo.Cursor, error) {
	return s.users.Aggregate(s.ctx, bson.A{
		bson.D{
			{Key: "$match",
				Value: bson.D{
					{Key: "rank",
						Value: bson.D{
							{Key: "$lte", Value: maxRank},
							{Key: "$ne", Value: 0},
						},
					},
				},
			},
		},
		bson.D{{Key: "$sample", Value: bson.D{{Key: "size", Value: max}}}},
	})
}

func (s *Store) DeleteUser(id string) error {
	filter := bson.D{{Key: "_id", Value: id}}
	if _, err := s.users.DeleteOne(s.ctx, filter); err != nil {
		return errors.Wrap(err, "deleting a user from mongo")
	}
	return nil
}
