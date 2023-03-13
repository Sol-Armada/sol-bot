package transaction

import (
	"context"
	"encoding/json"

	"github.com/apex/log"
	"github.com/sol-armada/admin/stores"
	"github.com/sol-armada/admin/user"
)

type Transaction struct {
	Id     string     `json:"_id" bson:"_id"`
	Amount int32      `json:"amount" bson:"amount"`
	From   *user.User `omitempty,json:"from" bson:"from"`
	To     *user.User `omitempty,json:"to" bson:"to"`
	Holder *user.User `json:"holder" bson:"holder"`
	Notes  string     `json:"notes" bson:"notes"`
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
	cur, err := stores.Storage.GetTransactions()
	if err != nil {
		return nil, err
	}

	if err := cur.All(context.Background(), &transactions); err != nil {
		return nil, err
	}

	return transactions, nil
}

func (t *Transaction) ToMap() map[string]interface{} {
	jsonEvent, err := json.Marshal(t)
	if err != nil {
		log.WithError(err).WithField("transaction", t).Error("transaction to json")
		return map[string]interface{}{}
	}

	var transactionMap map[string]interface{}
	if err := json.Unmarshal(jsonEvent, &transactionMap); err != nil {
		log.WithError(err).WithField("transaction", t).Error("transaction to map")
		return map[string]interface{}{}
	}

	return transactionMap
}

func (t *Transaction) Save() error {
	return stores.Storage.SaveTransaction(t.ToMap())
}
