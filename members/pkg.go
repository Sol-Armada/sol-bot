package members

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
	"github.com/sol-armada/admin/cache"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/ranks"
	"github.com/sol-armada/admin/stores"
	"go.mongodb.org/mongo-driver/bson"
)

type GameplayTypes string

const (
	BountyHunting  GameplayTypes = "bounty hunting"
	Engineering    GameplayTypes = "engineering"
	Exporation     GameplayTypes = "exporation"
	FpsCombat      GameplayTypes = "fps combat"
	Hauling        GameplayTypes = "hauling"
	Medical        GameplayTypes = "medical"
	Mining         GameplayTypes = "mining"
	Reconnaissance GameplayTypes = "reconnaissance"
	Racing         GameplayTypes = "racing"
	Scrapping      GameplayTypes = "scrapping"
	ShipCrew       GameplayTypes = "ship crew"
	ShipCombat     GameplayTypes = "ship combat"
	Trading        GameplayTypes = "trading"
)

type Member struct {
	Id             string     `json:"id" bson:"_id"`
	Name           string     `json:"name" bson:"name"`
	Rank           ranks.Rank `json:"rank" bson:"rank"`
	Notes          string     `json:"notes" bson:"notes"`
	Events         int64      `json:"events" bson:"events"`
	LegacyEvents   int64      `json:"legacy_events" bson:"legacy_events"`
	PrimaryOrg     string     `json:"primary_org" bson:"primary_org"`
	RSIMember      bool       `json:"rsi_member" bson:"rsi_member"`
	BadAffiliation bool       `json:"bad_affiliation" bson:"bad_affiliation"`
	Affilations    []string   `json:"affiliations" bson:"affilations"`
	Avatar         string     `json:"avatar" bson:"avatar"`
	Updated        time.Time  `json:"updated" bson:"updated"`
	IsBot          bool       `json:"is_bot" bson:"is_bot"`
	IsAlly         bool       `json:"is_ally" bson:"is_ally"`
	Validated      bool       `json:"validated" bson:"validated"`
	ValidationCode string     `json:"validation_code" bson:"validation_code"`

	Merits   []*Merit   `json:"merits" bson:"merits"`
	Demerits []*Demerit `json:"demerits" bson:"demerits"`

	// onboarding info
	Playtime  int             `json:"playtime" bson:"playtime"`
	Gameplay  []GameplayTypes `json:"gamplay" bson:"gameplay"`
	Age       int             `json:"age" bson:"age"`
	Recruiter *Member         `json:"recruiter" bson:"recruiter"`
}

type AttendedEvent struct {
	ID   string `json:"id" bson:"_id"`
	Name string `json:"name" bson:"name"`
}

type Merit struct {
	Giver  Member    `json:"giver" bson:"giver"`
	Reason string    `json:"reason" bson:"reason"`
	When   time.Time `json:"when" bson:"when"`
}

type Demerit struct {
	Giver  Member    `json:"giver" bson:"giver"`
	Reason string    `json:"reason" bson:"reason"`
	When   time.Time `json:"when" bson:"when"`
}

type MemberError error

var (
	MemberNotFound MemberError = errors.New("member not found")
)

func New(discordMember *discordgo.Member) *Member {
	u := &Member{
		Id:             discordMember.User.ID,
		Rank:           ranks.Guest,
		PrimaryOrg:     "",
		Notes:          "",
		Events:         0,
		RSIMember:      true,
		BadAffiliation: false,
		Avatar:         discordMember.Avatar,
	}

	u.Name = u.GetTrueNick(discordMember)

	return u
}

func Get(id string) (*Member, error) {
	member := &Member{}

	// check the cache
	rawUser := cache.Cache.GetUser(id)
	if rawUser != nil {
		userByte, _ := json.Marshal(rawUser)
		if err := json.Unmarshal(userByte, member); err != nil {
			return nil, err
		}
	}

	// check the store
	if err := stores.Users.Get(id).Decode(member); err != nil {
		return nil, err
	}
	return member, nil
}

func GetRandom(max int, maxRank int) ([]Member, error) {
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

	users := []Member{}
	for randomUsers.Next(stores.GetContext()) {
		member := Member{}
		if err := randomUsers.Decode(&member); err != nil {
			return nil, err
		}
		users = append(users, member)
	}

	return users, nil
}

func (u *Member) GetTrueNick(discordMember *discordgo.Member) string {
	if discordMember == nil {
		return u.Name
	}

	trueNick := discordMember.User.Username
	if discordMember.Nick != "" {
		regRank := regexp.MustCompile(`\[(.*?)\] `)
		regAlly := regexp.MustCompile(`\{(.*?)\} `)
		regPronouns := regexp.MustCompile(` \((.*?)\)`)
		trueNick = regRank.ReplaceAllString(discordMember.Nick, "")
		trueNick = regAlly.ReplaceAllString(trueNick, "")
		trueNick = regPronouns.ReplaceAllString(trueNick, "")
	}

	return trueNick
}

func (u *Member) Save() error {
	log.WithField("member", u).Debug("saving member")
	u.Updated = time.Now().UTC()
	cache.Cache.SetUser(u.Id, u.ToMap())

	return stores.Users.Update(u.Id, u)
}

func (u *Member) UpdateEventCount(count int64) {
	u.Events = count
	u.LegacyEvents = u.Events
	_ = u.Save()
}

func (u *Member) IncrementEventCount() {
	u.Events++
	u.LegacyEvents = u.Events
	_ = u.Save()
}

func (u *Member) DecrementEventCount() {
	u.Events--
	u.LegacyEvents = u.Events
	_ = u.Save()
}

func (u *Member) IsAdmin() bool {
	logger := log.WithField("id", u.Id)
	logger.Debug("checking if admin")
	if u.Rank <= ranks.Lieutenant {
		return true
	}
	logger.Debug("is NOT admin")
	return false
}

func (u *Member) Login(code string) error {
	log.WithField("access code", code).Debug("logging in")
	access, err := auth.Authenticate(code)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", "https://discord.com/api/v10/users/@me", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", access.Token))

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

	discordUser := &discordgo.Member{}
	if err := json.NewDecoder(resp.Body).Decode(&discordUser); err != nil {
		return err
	}

	if err := stores.Users.Get(discordUser.User.ID).Decode(&u); err != nil {
		return errors.Wrap(err, "getting stored member")
	}

	u.Avatar = discordUser.Avatar
	_ = u.Save()

	return nil
}

func (u *Member) Delete() error {
	log.WithField("member", u).Debug("deleting member")

	cache.Cache.DeleteUser(u.Id)

	return nil
}

func (u *Member) ToMap() map[string]interface{} {
	r := map[string]interface{}{}
	b, _ := json.Marshal(u)
	_ = json.Unmarshal(b, &r)
	return r
}

func (u *Member) Issues() []string {
	issues := []string{}

	if u.IsBot {
		issues = append(issues, "bot")
	}

	if u.Rank == ranks.Guest {
		issues = append(issues, "guest")
	}

	if u.Rank == ranks.Recruit && !u.RSIMember {
		issues = append(issues, "non-rsi member but is recruit")
	}

	if u.Rank == ranks.Ally {
		issues = append(issues, "ally")
	}

	if u.BadAffiliation {
		issues = append(issues, "bad affiliation")
	}

	if u.PrimaryOrg == "REDACTED" {
		issues = append(issues, "redacted org")
	}

	if u.Rank <= ranks.Member && u.PrimaryOrg != config.GetString("rsi_org_sid") {
		issues = append(issues, "bad primary org")
	}

	switch u.Rank {
	case ranks.Recruit:
		if u.Events >= 3 {
			issues = append(issues, "max event credits for this rank (3)")
		}
	case ranks.Member:
		if u.Events >= 10 {
			issues = append(issues, "max event credits for this rank (10)")
		}
	case ranks.Technician:
		if u.Events >= 20 {
			issues = append(issues, "max event credits for this rank (20)")
		}
	}

	return issues
}

func (u *Member) GiveMerit(reason string, who *Member) error {
	u.Merits = append(u.Merits, &Merit{
		Giver:  *who,
		Reason: reason,
		When:   time.Now().UTC(),
	})

	return u.Save()
}

func (u *Member) GiveDemerit(reason string, who *Member) error {
	u.Demerits = append(u.Demerits, &Demerit{
		Giver:  *who,
		Reason: reason,
		When:   time.Now().UTC(),
	})

	return u.Save()
}
