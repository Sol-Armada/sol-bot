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
		bson.D{
			{Key: "$match",
				Value: bson.D{
					{Key: "$and",
						Value: bson.A{
							bson.D{{Key: "recorded", Value: true}},
							bson.D{
								{Key: "members",
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
		bson.D{{"$sort", bson.D{{"date_created", 1}}}},
		bson.D{
			{"$group",
				bson.D{
					{"_id", primitive.Null{}},
					{"records",
						bson.D{
							{"$push",
								bson.D{
									{"_id", "$_id"},
									{"name", "$name"},
									{"date_created", "$date_created"},
								},
							},
						},
					},
				},
			},
		},
		bson.D{
			{"$addFields",
				bson.D{
					{"recordsWithPrev",
						bson.D{
							{"$map",
								bson.D{
									{"input",
										bson.D{
											{"$range",
												bson.A{
													0,
													bson.D{{"$size", "$records"}},
												},
											},
										},
									},
									{"as", "i"},
									{"in",
										bson.D{
											{"current",
												bson.D{
													{"$arrayElemAt",
														bson.A{
															"$records",
															"$$i",
														},
													},
												},
											},
											{"prev",
												bson.D{
													{"$switch",
														bson.D{
															{"branches",
																bson.A{
																	bson.D{
																		{"case",
																			bson.D{
																				{"$eq",
																					bson.A{
																						"$$i",
																						0,
																					},
																				},
																			},
																		},
																		{"then", primitive.Null{}},
																	},
																},
															},
															{"default",
																bson.D{
																	{"$arrayElemAt",
																		bson.A{
																			"$records",
																			bson.D{
																				{"$subtract",
																					bson.A{
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
					},
				},
			},
		},
		bson.D{{"$unwind", "$recordsWithPrev"}},
		bson.D{
			{"$addFields",
				bson.D{
					{"time_delta_hours",
						bson.D{
							{"$round",
								bson.A{
									bson.D{
										{"$divide",
											bson.A{
												bson.D{
													{"$subtract",
														bson.A{
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
		bson.D{{"$addFields", bson.D{{"date_created", "$recordsWithPrev.current.date_created"}}}},
		bson.D{{"$sort", bson.D{{"date_created", -1}}}},
		bson.D{
			{"$group",
				bson.D{
					{"_id",
						bson.D{
							{"$switch",
								bson.D{
									{"branches",
										bson.A{
											bson.D{
												{"case",
													bson.D{
														{"$and",
															bson.A{
																bson.D{
																	{"$lte",
																		bson.A{
																			"$time_delta_hours",
																			8,
																		},
																	},
																},
																bson.D{
																	{"$not",
																		bson.A{
																			bson.D{
																				{"$eq",
																					bson.A{
																						"$recordsWithPrev.prev",
																						primitive.Null{},
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
												{"then",
													bson.D{
														{"$concat",
															bson.A{
																bson.D{
																	{"$dateToString",
																		bson.D{
																			{"date", "$date_created"},
																			{"format", "%Y-%m-%d-%H"},
																		},
																	},
																},
																"-overlap",
															},
														},
													},
												},
											},
										},
									},
									{"default",
										bson.D{
											{"$dateToString",
												bson.D{
													{"date", "$date_created"},
													{"format", "%Y-%m-%d-%H"},
												},
											},
										},
									},
								},
							},
						},
					},
					{"names", bson.D{{"$push", "$recordsWithPrev.current.name"}}},
					{"ids", bson.D{{"$push", "$recordsWithPrev.current._id"}}},
					{"count", bson.D{{"$sum", 1}}},
				},
			},
		},
		bson.D{{"$sort", bson.D{{"_id", 1}}}},
		bson.D{{"$match", bson.D{{"_id", bson.D{{"$not", bson.D{{"$regex", "overlap"}}}}}}}},
		bson.D{{"$count", "count"}},
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
