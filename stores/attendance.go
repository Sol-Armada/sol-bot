package stores

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AttendanceStore struct {
	*store
}

func newAttendanceStore(ctx context.Context, client *mongo.Client, database string) *AttendanceStore {
	_ = client.Database(database).CreateCollection(ctx, string(ATTENDANCE))
	s := &store{
		Collection: client.Database(database).Collection(string(ATTENDANCE)),
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
					{Key: "localField", Value: "with_issues"},
					{Key: "foreignField", Value: "_id"},
					{Key: "as", Value: "with_issues"},
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
		bson.D{
			{Key: "$unwind",
				Value: bson.D{
					{Key: "path", Value: "$submitted_by"},
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
					{Key: "localField", Value: "with_issues"},
					{Key: "foreignField", Value: "_id"},
					{Key: "as", Value: "with_issues"},
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
		bson.D{
			{Key: "$unwind",
				Value: bson.D{
					{Key: "path", Value: "$submitted_by"},
					{Key: "includeArrayIndex", Value: "object"},
				},
			},
		},
		bson.D{{Key: "$match", Value: filter}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "date_created", Value: -1}}}},
	}

	if limit > 0 {
		if page == 0 {
			page = 1
		}

		page = (page - 1) * limit
		pipeline = append(pipeline, bson.D{{Key: "$skip", Value: page}}, bson.D{{Key: "$limit", Value: limit}})
	}

	cur, err := s.Aggregate(s.ctx, pipeline)
	if err != nil {
		return nil, err
	}

	return cur, nil
}

func (s *AttendanceStore) GetCount(memberId string) (int, error) {
	pipeline := bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "recorded", Value: true}}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "date_created", Value: 1}}}},
		bson.D{
			{Key: "$group",
				Value: bson.D{
					{Key: "_id", Value: primitive.Null{}},
					{Key: "records",
						Value: bson.D{
							{Key: "$push",
								Value: bson.D{
									{Key: "_id", Value: "$_id"},
									{Key: "name", Value: "$name"},
									{Key: "date_created", Value: "$date_created"},
									{Key: "members", Value: "$members"},
									{Key: "recorded", Value: "$recorded"},
								},
							},
						},
					},
				},
			},
		},
		bson.D{
			{Key: "$addFields",
				Value: bson.D{
					{Key: "recordsWithPrev",
						Value: bson.D{
							{Key: "$map",
								Value: bson.D{
									{Key: "input",
										Value: bson.D{
											{Key: "$range",
												Value: bson.A{
													1,
													bson.D{{Key: "$size", Value: "$records"}},
												},
											},
										},
									},
									{Key: "as", Value: "i"},
									{Key: "in",
										Value: bson.D{
											{Key: "current",
												Value: bson.D{
													{Key: "$arrayElemAt",
														Value: bson.A{
															"$records",
															"$$i",
														},
													},
												},
											},
											{Key: "prev",
												Value: bson.D{
													{Key: "$arrayElemAt",
														Value: bson.A{
															"$records",
															bson.D{
																{Key: "$subtract",
																	Value: bson.A{
																		"$$i",
																		1,
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		bson.D{{Key: "$unwind", Value: "$recordsWithPrev"}},
		bson.D{
			{Key: "$addFields",
				Value: bson.D{
					{Key: "time_delta_hours",
						Value: bson.D{
							{Key: "$round",
								Value: bson.A{
									bson.D{
										{Key: "$divide",
											Value: bson.A{
												bson.D{
													{Key: "$subtract",
														Value: bson.A{
															"$recordsWithPrev.current.date_created",
															"$recordsWithPrev.prev.date_created",
														},
													},
												},
												3600000,
											},
										},
									},
									0,
								},
							},
						},
					},
				},
			},
		},
		bson.D{{Key: "$addFields", Value: bson.D{{Key: "date_created", Value: "$recordsWithPrev.current.date_created"}}}},
		bson.D{
			{Key: "$match",
				Value: bson.D{
					{Key: "$and",
						Value: bson.A{
							bson.D{{Key: "recordsWithPrev.current.recorded", Value: true}},
							bson.D{
								{Key: "recordsWithPrev.current.members",
									Value: bson.D{
										{Key: "$in",
											Value: bson.A{
												memberId,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		bson.D{
			{Key: "$group",
				Value: bson.D{
					{Key: "_id",
						Value: bson.D{
							{Key: "$switch",
								Value: bson.D{
									{Key: "branches",
										Value: bson.A{
											bson.D{
												{Key: "case",
													Value: bson.D{
														{Key: "$lte",
															Value: bson.A{
																"$time_delta_hours",
																8,
															},
														},
													},
												},
												{Key: "then",
													Value: bson.D{
														{Key: "$concat",
															Value: bson.A{
																bson.D{
																	{Key: "$dateToString",
																		Value: bson.D{
																			{Key: "date", Value: "$date_created"},
																			{Key: "format", Value: "%Y-%m-%d-%H"},
																		},
																	},
																},
																"-delta-",
																bson.D{{Key: "$toString", Value: "$time_delta_hours"}},
															},
														},
													},
												},
											},
										},
									},
									{Key: "default",
										Value: bson.D{
											{Key: "$dateToString",
												Value: bson.D{
													{Key: "date", Value: "$date_created"},
													{Key: "format", Value: "%Y-%m-%d-%H"},
												},
											},
										},
									},
								},
							},
						},
					},
					{Key: "names", Value: bson.D{{Key: "$push", Value: "$recordsWithPrev.current.name"}}},
					{Key: "ids", Value: bson.D{{Key: "$push", Value: "$recordsWithPrev.current._id"}}},
					{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
				},
			},
		},
		bson.D{{Key: "$count", Value: "count"}},
	}

	cur, err := s.Aggregate(s.ctx, pipeline)
	if err != nil {
		return 0, err
	}

	// get the count
	var result bson.M
	cur.Next(s.ctx)

	if err := cur.Decode(&result); err != nil {
		if err.Error() == "EOF" {
			return 0, nil
		}

		return 0, err
	}

	return int(result["count"].(int32)), nil
}

func (s *AttendanceStore) Upsert(id string, attendance any) error {
	_, err := s.UpdateOne(s.ctx, bson.M{"_id": id}, bson.M{"$set": attendance}, options.Update().SetUpsert(true))
	return err
}

func (s *AttendanceStore) Delete(id string) error {
	_, err := s.DeleteOne(s.ctx, bson.M{"_id": id})
	return err
}
