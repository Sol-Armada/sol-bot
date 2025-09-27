package stores

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type KanbanStore struct {
	*store
}

const KANBAN Collection = "kanban"

func newKanbanStore(ctx context.Context, client *mongo.Client, database string) *KanbanStore {
	_ = client.Database(database).CreateCollection(ctx, string(KANBAN))
	s := &store{
		Collection: client.Database(database).Collection(string(KANBAN)),
		ctx:        ctx,
	}
	return &KanbanStore{s}
}

func (c *Client) GetKanbanStore() (*KanbanStore, bool) {
	storeInterface, ok := c.GetCollection(KANBAN)
	if !ok {
		return nil, false
	}
	return storeInterface.(*KanbanStore), ok
}

func (k *KanbanStore) Get(id string) ([]byte, error) {
	cur, err := k.Aggregate(k.ctx, bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "_id", Value: id}}}},
		bson.D{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "members"},
			{Key: "localField", Value: "assignee"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "assignee"},
		}}},
		bson.D{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$assignee"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}},
		bson.D{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "members"},
			{Key: "localField", Value: "created_by"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "created_by"},
		}}},
		bson.D{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$created_by"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}},
	})
	if err != nil {
		return nil, err
	}
	defer cur.Close(k.ctx)
	if !cur.Next(k.ctx) {
		return nil, mongo.ErrNoDocuments
	}
	return cur.Current, nil
}

func (k *KanbanStore) Upsert(id string, card any) error {
	opts := options.Replace().SetUpsert(true)
	_, err := k.Collection.ReplaceOne(k.ctx, bson.D{{Key: "_id", Value: id}}, card, opts)
	return err
}
