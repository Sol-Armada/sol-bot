package stores

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AttendanceStore struct {
	*store
}

func newAttendanceStore(ctx context.Context, client *mongo.Client, database string) *AttendanceStore {
	s := &store{
		Collection: client.Database(database).Collection("attendance"),
		ctx:        ctx,
	}
	return &AttendanceStore{s}
}

func (s *AttendanceStore) Create(attendance any) error {
	_, err := s.InsertOne(s.ctx, attendance)
	return err
}

func (s *AttendanceStore) Get(id string) (*mongo.Cursor, error) {
	pipeline := bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "_id", Value: id}}}},
		bson.D{
			{Key: "$lookup",
				Value: bson.D{
					{Key: "from", Value: "members"},
					{Key: "localField", Value: "issues"},
					{Key: "foreignField", Value: "_id"},
					{Key: "as", Value: "issues"},
				},
			},
		},
		bson.D{
			{Key: "$lookup",
				Value: bson.D{
					{Key: "from", Value: "members"},
					{Key: "localField", Value: "members"},
					{Key: "foreignField", Value: "_id"},
					{Key: "as", Value: "members"},
				},
			},
		},
	}

	cur, err := s.Aggregate(s.ctx, pipeline)
	if err != nil {
		return nil, err
	}

	return cur, nil
}

// List retrieves a list of attendance records from the database, optionally filtered by the provided filter and limited to the specified number of records.
//
// Parameters:
// - filter: An interface{} representing the filter to apply to the query.
// - limit: An int64 representing the maximum number of records to retrieve. If limit is 0, all records will be retrieved.
//
// Returns:
// - *mongo.Cursor: A cursor to iterate over the retrieved attendance records.
// - error: An error if the query operation fails.
func (s *AttendanceStore) List(filter interface{}, limit int, page int) (*mongo.Cursor, error) {
	pipeline := bson.A{
		bson.D{
			{Key: "$lookup",
				Value: bson.D{
					{Key: "from", Value: "members"},
					{Key: "localField", Value: "issues"},
					{Key: "foreignField", Value: "_id"},
					{Key: "as", Value: "issues"},
				},
			},
		},
		bson.D{
			{Key: "$lookup",
				Value: bson.D{
					{Key: "from", Value: "members"},
					{Key: "localField", Value: "members"},
					{Key: "foreignField", Value: "_id"},
					{Key: "as", Value: "members"},
				},
			},
		},
		bson.D{
			{Key: "$lookup",
				Value: bson.D{
					{Key: "from", Value: "members"},
					{Key: "localField", Value: "submitted_by"},
					{Key: "foreignField", Value: "_id"},
					{Key: "as", Value: "submitted_by"},
				},
			},
		},
		bson.D{{Key: "$match", Value: filter}},
	}

	if limit > 0 {
		if page == 0 {
			page = 1
		}

		page = (page - 1) * limit
		pipeline = append(pipeline, bson.D{{Key: "$limit", Value: limit}}, bson.D{{Key: "$skip", Value: page}})
	}

	cur, err := s.Aggregate(s.ctx, pipeline)
	if err != nil {
		return nil, err
	}

	return cur, nil
}

func (s *AttendanceStore) Upsert(id string, attendance any) error {
	_, err := s.UpdateOne(s.ctx, bson.M{"_id": id}, bson.M{"$set": attendance}, options.Update().SetUpsert(true))
	return err
}

func (s *AttendanceStore) Delete(id string) error {
	_, err := s.DeleteOne(s.ctx, bson.M{"_id": id})
	return err
}
