package request

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/stores"
	"github.com/sol-armada/sol-bot/users"
)

func GetUser(r *http.Request) (*users.User, error) {
	vars := mux.Vars(r)
	if vars["id"] == "" {
		return nil, errors.New("missing user id")
	}

	storedUser := &users.User{}
	if err := stores.Users.Get(vars["id"]).Decode(&storedUser); err != nil {
		return nil, errors.Wrap(err, "getting user from request")
	}

	return storedUser, nil
}

func GetBody(r *http.Request) (map[string]interface{}, error) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrap(err, "reading body")
	}
	body := map[string]interface{}{}
	if err := json.Unmarshal(b, &body); err != nil {
		return nil, errors.Wrap(err, "unmarshalling body")
	}

	return body, nil
}
