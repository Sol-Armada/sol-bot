package users

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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type User struct {
	ID             string     `json:"id" bson:"_id"`
	Name           string     `json:"name" bson:"name"`
	Rank           ranks.Rank `json:"rank" bson:"rank"`
	Notes          string     `json:"notes" bson:"notes"`
	Events         int64      `json:"events" bson:"events"`
	PrimaryOrg     string     `json:"primary_org" bson:"primary_org"`
	RSIMember      bool       `json:"rsi_member" bson:"rsi_member"`
	BadAffiliation bool       `json:"bad_affiliation" bson:"bad_affiliation"`
	Affilations    []string   `json:"affiliations" bson:"affilations"`
	Avatar         string     `json:"avatar" bson:"avatar"`
	Updated        time.Time  `json:"updated" bson:"updated"`
	IsBot          bool       `json:"is_bot" bson:"is_bot"`
	IsAlly         bool       `json:"is_ally" bson:"is_ally"`
	Validated      bool       `json:"validated" bson:"validated"`

	Playtime  string `json:"playtime" bson:"playtime"`
	Gameplay  string `json:"gamplay" bson:"gameplay"`
	Age       int    `json:"age" bson:"age"`
	Recruiter *User  `json:"recruiter" bson:"recruiter"`

	Discord *discordgo.Member `json:"-" bson:"discord"`
	Access  *auth.UserAccess  `json:"-" bson:"access"`
}

func New(m *discordgo.Member) *User {
	name := m.User.Username
	if m.Nick != "" {
		name = m.Nick
	}
	u := &User{
		ID:             m.User.ID,
		Name:           name,
		Rank:           ranks.Guest,
		PrimaryOrg:     "",
		Notes:          "",
		Events:         0,
		RSIMember:      true,
		Discord:        m,
		BadAffiliation: false,
		Avatar:         m.User.Avatar,
	}
	u.Name = u.GetTrueNick()

	return u
}

func Get(id string) (*User, error) {
	result := stores.Users.Get(id)
	user := &User{}
	if err := result.Decode(&user); err != nil {
		return nil, err
	}
	return user, nil
}

func GetRandom(max int, maxRank int) ([]User, error) {
	stores := stores.Users
	randomUsers, err := stores.Aggregate(stores.GetContext(), bson.A{
		bson.D{
			{Key: "$match",
				Value: bson.D{
					{Key: "rank",
						Value: bson.D{
							{Key: "$lte", Value: maxRank},
							{Key: "$ne", Value: 0},
						},
					},
				},
			},
		},
		bson.D{{Key: "$sample", Value: bson.D{{Key: "size", Value: max}}}},
	})
	if err != nil {
		return nil, err
	}

	users := []User{}
	for randomUsers.Next(stores.GetContext()) {
		user := User{}
		if err := randomUsers.Decode(&user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (u *User) GetTrueNick() string {
	if u.Discord == nil {
		return u.Name
	}

	trueNick := u.Discord.User.Username
	regRank := regexp.MustCompile(`\[(.*?)\] `)
	regAlly := regexp.MustCompile(`\{(.*?)\} `)
	regPronouns := regexp.MustCompile(` \((.*?)\)`)
	if u.Discord.Nick != "" {
		trueNick = regRank.ReplaceAllString(u.Discord.Nick, "")
		trueNick = regAlly.ReplaceAllString(trueNick, "")
		trueNick = regPronouns.ReplaceAllString(trueNick, "")
	}

	return trueNick
}

func (u *User) Save() error {
	log.WithField("user", u).Debug("saving user")
	stores := stores.Users

	u.Updated = time.Now().UTC()

	opts := options.Replace().SetUpsert(true)
	if _, err := stores.ReplaceOne(stores.GetContext(), bson.D{{Key: "_id", Value: u.ID}}, u, opts); err != nil {
		return errors.Wrap(err, "saving user to mongo")
	}

	return nil
}

func (u *User) Load() error {
	log.WithField("user", u).Debug("loading user")
	if err := stores.Users.Get(u.Discord.User.ID).Decode(u); err != nil {
		return errors.Wrap(err, "getting stored user for loading")
	}

	return nil
}

func (u *User) UpdateEventCount(count int64) error {
	u.Events = count
	return u.Save()
}

func (u *User) IsAdmin() bool {
	logger := log.WithField("id", u.ID)
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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", u.Access.AccessToken))

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
		return errors.New(string(errorMessage))
	}

	discordUser := &discordgo.User{}
	if err := json.NewDecoder(resp.Body).Decode(&discordUser); err != nil {
		return err
	}

	if err := stores.Users.Get(discordUser.ID).Decode(&u); err != nil {
		return errors.Wrap(err, "getting stored user")
	}

	u.Avatar = discordUser.Avatar
	if err := u.Save(); err != nil {
		return errors.Wrap(err, "saving stored user")
	}

	return nil
}

func (u *User) StillLoggedIn() bool {
	return time.Until(u.Access.ExpiresAt) > 0
}

func (u *User) Delete() error {
	log.WithField("user", u).Debug("deleting user")
	store := stores.Users

	if _, err := store.DeleteOne(store.GetContext(), bson.M{"_id": u.ID}); err != nil {
		return errors.Wrap(err, "deleting user")
	}
	return nil
}
