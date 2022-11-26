package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/apex/log"
	"github.com/gorilla/mux"
	"github.com/sol-armada/admin/ranks"
	"github.com/sol-armada/admin/stores"
	"github.com/sol-armada/admin/users"
)

func Login(w http.ResponseWriter, r *http.Request) {
	// make sure we are only getting post
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// extract the body for the code
	body := map[string]interface{}{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.WithError(err).Error("extracting body from request")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// make sure the code is real
	code, ok := body["code"].(string)
	if !ok {
		log.Error("body does not have the code")
		http.Error(w, "Invalid Parameters", http.StatusBadRequest)
		return
	}

	// create the user
	u := &users.User{
		ID:         "",
		Rank:       ranks.Recruit,
		Ally:       false,
		PrimaryOrg: "",
		Notes:      "",
		Events:     0,
		RSIMember:  true,
		Discord:    nil,
	}

	if err := u.Login(code); err != nil {
		log.WithError(err).Error("authenicating user")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// check the user is allowed
	if !u.IsAdmin() {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if _, err := fmt.Fprint(w, u.ToJson()); err != nil {
		log.WithError(err).Error("sending login response")
	}
}

func CheckLogin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	storedUser := &users.User{}
	if err := stores.Storage.GetUser(id).Decode(storedUser); err != nil {
		log.WithError(err).Error("check login return")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := fmt.Fprint(w, storedUser.StillLoggedIn()); err != nil {
		log.WithError(err).Error("check login return")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func EncryptAccess(w http.ResponseWriter, r *http.Request) {}

func DecryptAccess(w http.ResponseWriter, r *http.Request) {}
