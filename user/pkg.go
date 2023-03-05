package user

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/admin/auth"
	"github.com/sol-armada/admin/ranks"
	"github.com/sol-armada/admin/stores"
)

type User struct {
	ID             string            `json:"id" bson:"_id"`
	Name           string            `json:"name" bson:"name"`
	Rank           ranks.Rank        `json:"rank" bson:"rank"`
	Notes          string            `json:"notes" bson:"notes"`
	Events         int64             `json:"events" bson:"events"`
	PrimaryOrg     string            `json:"primary_org" bson:"primary_org"`
	RSIMember      bool              `json:"rsi_member" bson:"rsi_member"`
	BadAffiliation bool              `json:"bad_affiliation" bson:"bad_affiliation"`
	Discord        *discordgo.Member `json:"discord" bson:"discord"`

	Access *auth.UserAccess `json:"access" bson:"access"`
}

func New(m *discordgo.Member) *User {
	name := m.User.Username
	if m.Nick != "" {
		name = m.Nick
	}
	u := &User{
		ID:         m.User.ID,
		Name:       name,
		Rank:       ranks.Guest,
		PrimaryOrg: "",
		Notes:      "",
		Events:     0,
		RSIMember:  true,
		Discord:    m,
	}

	return u
}

func (u *User) GetTrueNick() string {
	trueNick := u.Discord.User.Username
	regRank := regexp.MustCompile(`\[(.*?)\] `)
	regAlly := regexp.MustCompile(`\{(.*?)\} `)
	if u.Discord.Nick != "" {
		trueNick = regRank.ReplaceAllString(u.Discord.Nick, "")
		trueNick = regAlly.ReplaceAllString(trueNick, "")
	}

	return trueNick
}

func (u *User) Save() error {
	log.WithField("user", u.ToJson()).Debug("saving user")
	return stores.Storage.SaveUser(u.ID, u)
}

func (u *User) Load() error {
	log.WithField("user", u.ToJson()).Debug("loading user")
	if err := stores.Storage.GetUser(u.Discord.User.ID).Decode(u); err != nil {
		return errors.Wrap(err, "getting stored user for loading")
	}

	return nil
}

func (u *User) UpdateEventCount(count int64) error {
	u.Events = count
	return u.Save()
}

func (u *User) IsAdmin() bool {
	logger := log.WithField("id", u.Discord.User.ID)
	logger.Debug("checking if admin")
	if u.Rank <= ranks.Lieutenant {
		return true
	}
	logger.Debug("is NOT admin")
	return false
}

func (u *User) Login(code string) error {
	log.WithField("access code", code).Debug("logging in")
	userAccess, err := auth.Authenticate(code)
	if err != nil {
		return err
	}

	u.Access = userAccess

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

	loggedInUser := discordgo.User{}
	if err := json.NewDecoder(resp.Body).Decode(&loggedInUser); err != nil {
		return err
	}

	// id, ok := loggedInUser["id"].(string)
	// if !ok {
	// 	return errors.New("logged in user did not have an ID")
	// }

	storedUser := &User{}
	if err := stores.Storage.GetUser(loggedInUser.ID).Decode(&storedUser); err != nil {
		return errors.Wrap(err, "getting stored user")
	}

	if storedUser != nil {
		u.Update(storedUser)
	}

	return nil
}

func (u *User) StillLoggedIn() bool {
	return time.Until(u.Access.ExpiresAt) > 0
}

func (u *User) ToJson() string {
	jsonUser, err := json.Marshal(u)
	if err != nil {
		log.WithError(err).WithField("user", u).Error("user to json")
		return ""
	}
	return string(jsonUser)
}

func (u *User) Update(ui *User) {
	u.ID = ui.ID
	u.Name = ui.Name
	u.Rank = ui.Rank
	u.Notes = ui.Notes
	u.Events = ui.Events
	u.PrimaryOrg = ui.PrimaryOrg
	u.Discord = ui.Discord
}
