package request

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/users"
)

func GetUser(r *http.Request) (*users.User, error) {
	vars := mux.Vars(r)
	if vars["id"] == "" {
		return nil, errors.New("missing user id")
	}

	return users.GetUser(vars["id"])
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
