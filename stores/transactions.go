package stores

import (
	"github.com/pkg/errors"
	apierrors "github.com/sol-armada/admin/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Store) GetTransactions() (*mongo.Cursor, error) {
	return s.transactions.Find(s.ctx, bson.M{})
}

func (s *Store) SaveTransaction(t map[string]interface{}) error {
	id, ok := t["_id"].(string)
	if !ok {
		return apierrors.ErrMissingId
	}

	opts := options.Replace().SetUpsert(true)
	if _, err := s.transactions.ReplaceOne(s.ctx, bson.D{{Key: "_id", Value: id}}, t, opts); err != nil {
		return errors.Wrap(err, "saving event")
	}

	return nil
}
