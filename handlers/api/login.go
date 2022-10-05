package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/apex/log"
	"github.com/gorilla/mux"
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
	if _, ok := body["code"].(string); !ok {
		log.Error("body does not have the code")
		http.Error(w, "Invalid Parameters", http.StatusBadRequest)
		return
	}

	// create the user
	admin, err := users.NewAdmin(body["code"].(string))
	if err != nil {
		if err.Error() == "Unauthorized" {
			http.Error(w, "Problem with getting this user from Discord: Unauthorized", http.StatusUnauthorized)
			return
		}

		log.WithError(err).Error("making new user")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// check the user is an admin
	if !users.IsAdmin(admin.User.Id) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// store the admin
	admin.StoreAsAdmin()

	// convert the admin to json
	adminJson, err := json.Marshal(admin)
	if err != nil {
		log.WithError(err).Error("converting user to json")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := fmt.Fprint(w, string(adminJson)); err != nil {
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

	admin := users.GetAdmin(id)
	if _, err := fmt.Fprint(w, admin.StillLoggedIn()); err != nil {
		log.WithError(err).Error("check login return")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func EncryptAccess(w http.ResponseWriter, r *http.Request) {}

func DecryptAccess(w http.ResponseWriter, r *http.Request) {}
