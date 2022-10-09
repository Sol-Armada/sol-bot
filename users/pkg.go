package users

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/apex/log"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/config"
)

type Admin struct {
	User   *User       `json:"user"`
	Access *UserAccess `json:"access"`
}

type User struct {
	Id            string `json:"id"`
	Username      string `json:"username"`
	Nick          string `json:"nick"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`
	Rank          Rank   `json:"rank"`
	Ally          bool   `json:"ally"`
	Notes         string `json:"notes"`
	Events        int64  `json:"events"`
	PrimaryOrg    string `json:"primary_org"`
	RSIMember     bool   `json:"rsi_member"`
}

var Admins map[string]*Admin = map[string]*Admin{}

func LoadAdmins() error {
	log.Debug("loading admins")

	admins, err := storage.GetAdmins()
	if err != nil {
		return errors.Wrap(err, "get admins for loading")
	}

	Admins = admins
	return nil
}

func NewAdmin(code string) (*Admin, error) {
	log.WithField("code", code).Debug("creating new admin")

	admin := &Admin{}

	if err := admin.Login(code); err != nil {
		return nil, err
	}

	log.WithField("user", admin).Debug("successfully logged in")

	return admin, nil
}

func GetAdmin(id string) *Admin {
	if admin, ok := Admins[id]; ok {
		return admin
	}

	return nil
}

func (u *User) Save() error {
	log.WithField("user", u).Debug("saving user")
	return storage.SaveUser(u)
}

func (u *User) UpdateEventCount(count int64) error {
	u.Events = count
	return u.Save()
}

func (a *Admin) StoreAsAdmin() {
	Admins[a.User.Id] = a

	if err := storage.SaveAdmin(a); err != nil {
		log.WithError(err).Error("storing admin")
	}
}

func IsAdmin(id string) bool {
	logger := log.WithField("id", id)
	logger.Debug("checking if admin")
	whiteList := config.GetIntSlice("ADMINS")
	for _, uid := range whiteList {
		if id == strconv.Itoa(uid) {
			logger.Debug("is admin")
			return true
		}
	}

	logger.Debug("is NOT admin")
	return false
}

func (a *Admin) Login(code string) error {
	log.WithField("access code", code).Debug("logging in")
	userAccess, err := authenticate(code)
	if err != nil {
		return err
	}

	a.Access = userAccess

	req, err := http.NewRequest("GET", "https://discord.com/api/v10/users/@me", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", userAccess.AccessToken))

	log.Debug("request for login to discord")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if resp.StatusCode == 401 {
			return errors.New("Unauthorized")
		}
		errorMessage, _ := ioutil.ReadAll(resp.Body)
		log.WithField("message", errorMessage).Error(string(errorMessage))
		return errors.New("Internal Error")
	}

	if err := json.NewDecoder(resp.Body).Decode(&a.User); err != nil {
		return err
	}

	store := GetStorage()
	storedUser, err := store.GetUser(a.User.Id)
	if err != nil {
		return err
	}

	if storedUser != nil {
		a.User = storedUser
		return nil
	}

	return nil
}

func (a *Admin) StillLoggedIn() bool {
	return time.Until(a.Access.ExpiresAt) > 0
}

func GetUsers() ([]*User, error) {
	log.Debug("getting users")
	cachedUsers, err := storage.GetUsers()
	if err != nil {
		return nil, err
	}

	return cachedUsers, nil
}

func GetUser(id string) (*User, error) {
	cachedUser, err := storage.GetUser(id)
	if err != nil {
		return nil, errors.Wrap(err, "getting cached user")
	}

	return cachedUser, nil
}
