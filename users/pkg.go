package users

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/apex/log"
	"github.com/sol-armada/admin/config"
)

type User struct {
	Id            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`

	Access *UserAccess `json:"-"`
}

var Users map[string]*User = map[string]*User{}

func New(code string) (*User, error) {
	log.WithField("code", code).Debug("creating new user")
	userAccess, err := authenticate(code)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", "https://discord.com/api/v10/users/@me", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", userAccess.AccessToken))
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if resp.StatusCode == 401 {
			return nil, errors.New("Unauthorized")
		}
	}

	user := &User{}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return user, nil
}

func GetUser(id string) *User {
	if user, ok := Users[id]; ok {
		return user
	}

	return nil
}

func (u *User) Store() {
	Users[u.Id] = u
}

func (u *User) ToJson() (string, error) {
	userJson, err := json.Marshal(u)
	if err != nil {
		return "", err
	}

	return string(userJson), nil
}

func (u *User) IsAdmin() bool {
	log.WithField("user", u).Debug("checking if admin")
	whiteList := config.GetIntSlice("ADMINS")
	for _, uid := range whiteList {
		if u.Id == strconv.Itoa(uid) {
			log.WithField("user", u).Debug("is admin")
			return true
		}
	}

	log.WithField("user", u).Debug("is NOT admin")
	return false
}
