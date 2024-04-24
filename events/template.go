package events

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/stores"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Template struct {
	Id          string      `json:"id" bson:"_id"`
	Name        string      `json:"name" bson:"name"`
	AutoStart   bool        `json:"auto_start" bson:"auto_start"`
	Description string      `json:"description" bson:"description"`
	Cover       string      `json:"cover" bson:"cover"`
	Positions   []*Position `json:"positions" bson:"positions"`
	CommsTier   CommsTier   `json:"comms_tier" bson:"comms_tier"`
}

func TemplateExists(name string) (bool, error) {
	cur, err := stores.Templates.List(
		bson.D{{Key: "name", Value: name}},
	)
	if err != nil {
		return false, err
	}

	templates := []*Template{}
	if err := cur.All(context.Background(), &templates); err != nil {
		return false, err
	}

	for _, template := range templates {
		if template.Name == name {
			return true, nil
		}
	}

	return false, nil
}

func GetAllTemplates() ([]*Template, error) {
	cur, err := stores.Templates.List(bson.D{})
	if err != nil {
		return nil, err
	}

	templates := []*Template{}
	if err := cur.All(context.Background(), &templates); err != nil {
		return nil, err
	}

	return templates, nil
}

func (t *Template) Delete() error {
	return stores.Templates.Delete(t.Name)
}

func (t *Template) Save() error {
	store := stores.Templates

	opts := options.Replace().SetUpsert(true)
	if _, err := store.ReplaceOne(store.GetContext(), bson.D{{Key: "name", Value: t.Name}}, t, opts); err != nil {
		return errors.Wrap(err, "saving event")
	}

	return nil
}
