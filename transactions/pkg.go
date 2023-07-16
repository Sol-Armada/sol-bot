package transactions

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sol-armada/admin/stores"
	"github.com/sol-armada/admin/users"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Transaction struct {
	Id     string      `json:"id" bson:"_id"`
	Amount int32       `json:"amount" bson:"amount"`
	From   *users.User `omitempty,json:"from" bson:"from"`
	To     *users.User `omitempty,json:"to" bson:"to"`
	For    string      `omitempty,json:"for" bson:"for"`
	Holder *users.User `json:"holder" bson:"holder"`
	Notes  string      `json:"notes" bson:"notes"`
}

func New(body map[string]interface{}) (*Transaction, error) {
	bodyJson, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	transaction := &Transaction{}
	if err := json.Unmarshal(bodyJson, &transaction); err != nil {
		return nil, err
	}

	return transaction, nil
}

func List() ([]*Transaction, error) {
	transactions := []*Transaction{}
	cur, err := stores.Transactions.List(bson.D{})
	if err != nil {
		return nil, err
	}

	if err := cur.All(context.Background(), &transactions); err != nil {
		return nil, err
	}

	return transactions, nil
}

func (t *Transaction) Save() error {
	store := stores.Transactions

	opts := options.Replace().SetUpsert(true)
	if _, err := store.ReplaceOne(store.GetContext(), bson.D{{Key: "_id", Value: t.Id}}, t, opts); err != nil {
		return errors.Wrap(err, "saving transaction")
	}

	return nil
}
