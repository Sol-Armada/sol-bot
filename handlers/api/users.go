package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/apex/log"
	"github.com/sol-armada/admin/request"
	"github.com/sol-armada/admin/users"
)

func GetUsers(w http.ResponseWriter, r *http.Request) {
	logger := log.WithFields(log.Fields{
		"endpoint": "GetUsers",
	})

	// make sure we are only getting get
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u, err := users.GetUsers()
	if err != nil {
		logger.WithError(err).Error("getting users")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	usersJson, err := json.Marshal(u)
	if err != nil {
		logger.WithError(err).Error("getting users")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	usersJsons := string(usersJson)

	if _, err := fmt.Fprint(w, usersJsons); err != nil {
		logger.WithError(err).Error("converting users to json")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func User(w http.ResponseWriter, r *http.Request) {
	logger := log.WithFields(log.Fields{
		"endpoint": "GetUser",
	})
	switch r.Method {
	case http.MethodGet:
		if _, err := fmt.Fprint(w, http.StatusNotImplemented); err != nil {
			logger.WithError(err).Error("returning status")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	case http.MethodPut:
		UpdateUser(w, r)
	}
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	logger := log.WithFields(log.Fields{
		"endpoint": "GetUser",
	})

	user, err := request.GetUser(r)
	if err != nil {
		log.WithError(err).Error("getting user")
	}

	userJson, err := json.Marshal(user)
	if err != nil {
		logger.WithError(err).Error("marshaling user")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := fmt.Fprint(w, userJson); err != nil {
		logger.WithError(err).Error("converting users to json")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func SetRank(w http.ResponseWriter, r *http.Request) {
	logger := log.WithFields(log.Fields{
		"endpoint": "SetRank",
	})

	// make sure we are only getting put
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := request.GetUser(r)
	if err != nil {
		logger.WithError(err).Error("getting user")
	}

	params, err := request.GetBody(r)
	if err != nil {
		logger.WithError(err).Error("getting body")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if params["rank"] == nil {
		http.Error(w, "Invalid Request", http.StatusBadRequest)
		return
	}

	rid, err := strconv.Atoi(params["rank"].(string))
	if err != nil {
		logger.WithError(err).Error("converting the rank id")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	user.Rank = users.Rank(rid)

	if err := user.Save(); err != nil {
		logger.WithError(err).Error("updating user")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := fmt.Fprint(w, http.StatusOK); err != nil {
		logger.WithError(err).Error("returning status")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func UpdateUser(w http.ResponseWriter, r *http.Request) {
	logger := log.WithFields(log.Fields{
		"endpoint": "UpdateUser",
	})

	params, err := request.GetBody(r)
	if err != nil {
		logger.WithError(err).Error("getting body")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	mu, err := json.Marshal(params["user"].(map[string]interface{}))
	if err != nil {
		logger.WithError(err).Error("marshal user from request")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	u := &users.User{}
	if err := json.Unmarshal(mu, u); err != nil {
		logger.WithError(err).Error("unmarshal user from request")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if u.Events < 0 {
		u.Events = 0
	}

	if err := u.Save(); err != nil {
		logger.WithError(err).Error("returning status")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	userJson, err := json.Marshal(u)
	if err != nil {
		logger.WithError(err).Error("marshal updated user")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := fmt.Fprint(w, string(userJson)); err != nil {
		logger.WithError(err).Error("returning status")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
