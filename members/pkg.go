package members

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sol-armada/sol-bot/auth"
	"github.com/sol-armada/sol-bot/ranks"
	"github.com/sol-armada/sol-bot/stores"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	LegacyEvents   int        `json:"legacy_events" bson:"legacy_events"`
	PrimaryOrg     string     `json:"primary_org" bson:"primary_org"`
	RSIMember      bool       `json:"rsi_member" bson:"rsi_member"`
	BadAffiliation bool       `json:"bad_affiliation" bson:"bad_affiliation"`
	Affilations    []string   `json:"affiliations" bson:"affilations"`
	Avatar         string     `json:"avatar" bson:"avatar"`
	Updated        time.Time  `json:"updated" bson:"updated"`
	Validated      bool       `json:"validated" bson:"validated"`
	ValidationCode string     `json:"validation_code" bson:"validation_code"`
	Joined         time.Time  `json:"joined" bson:"joined"`
	Suffix         string     `json:"suffix" bson:"suffix"`
	Onboarded      bool       `json:"onboarded" bson:"onboarded"`

	IsBot       bool `json:"is_bot" bson:"is_bot"`
	IsAlly      bool `json:"is_ally" bson:"is_ally"`
	IsAffiliate bool `json:"is_affiliate" bson:"is_affiliate"`
	IsGuest     bool `json:"is_guest" bson:"is_guest"`

	Merits   []*Merit   `json:"merits" bson:"merits"`
	Demerits []*Demerit `json:"demerits" bson:"demerits"`

	// onboarding info
	OnboardedAt *time.Time      `json:"onboarded_at" bson:"onboarded_at"`
	Age         int             `json:"age" bson:"age"`
	Pronouns    string          `json:"pronouns" bson:"pronouns"`
	Playtime    int             `json:"playtime" bson:"playtime"`
	Gameplay    []GameplayTypes `json:"gamplay" bson:"gameplay"`
	Recruiter   *Member         `json:"recruiter" bson:"recruiter"`
	ChannelId   string          `json:"channel_id" bson:"channel_id"`
	MessageId   string          `json:"message_id" bson:"message_id"`
	LeftAt      *time.Time      `json:"left_at" bson:"left_at"`

	LegacyAge       string `json:"legacy_age" bson:"legacy_age"`
	LegacyPlaytime  string `json:"legacy_playtime" bson:"legacy_playtime"`
	LegacyGamplay   string `json:"legacy_gamplay" bson:"legacy_gamplay"`
	LegacyRecruiter string `json:"legacy_recruiter" bson:"legacy_recruiter"`
	LegacyOther     string `json:"legacy_other" bson:"legacy_other"`
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

var membersStore *stores.MembersStore

func Setup() error {
	storesClient := stores.Get()
	ms, ok := storesClient.GetMembersStore()
	if !ok {
		return errors.New("members store not found")
	}
	membersStore = ms

	return nil
}

func New(discordMember *discordgo.Member) *Member {
	m := &Member{
		Id:      discordMember.User.ID,
		Avatar:  discordMember.Avatar,
		Joined:  discordMember.JoinedAt.UTC(),
		IsGuest: true,
	}

	m.Name = m.GetTrueNick(discordMember)

	return m
}

func Get(id string) (*Member, error) {
	member := &Member{}

	// check the store
	if err := membersStore.Get(id).Decode(member); err != nil {
		return nil, err
	}
	return member, nil
}

func GetRandom(max int, maxRank ranks.Rank) ([]Member, error) {
	membersMap, err := membersStore.GetRandom(max, int(maxRank))
	if err != nil {
		return nil, err
	}

	members := []Member{}

	for _, memberMap := range membersMap {
		j, _ := json.Marshal(memberMap)
		member := &Member{}
		if err := json.Unmarshal(j, member); err != nil {
			return nil, err
		}
		members = append(members, *member)
	}

	return members, nil
}

func List(page int) ([]Member, error) {
	opts := options.Find().SetSort(bson.D{
		{
			Key:   "rank",
			Value: 1,
		},
		{
			Key:   "name",
			Value: 1,
		},
	}).SetLimit(100).SetSkip(int64(100 * (page - 1)))

	cur, err := membersStore.List(bson.D{{Key: "is_bot", Value: bson.D{{Key: "$eq", Value: false}}}}, opts)
	if err != nil {
		return nil, err
	}

	members := []Member{}

	for cur.Next(context.Background()) {
		member := Member{}
		if err := cur.Decode(&member); err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return members, nil
}

func (m *Member) GetTrueNick(discordMember *discordgo.Member) string {
	if discordMember == nil {
		return m.Name
	}

	trueNick := discordMember.User.Username
	if discordMember.Nick != "" {
		regRank := regexp.MustCompile(`\[(.*?)\] `)
		regAlly := regexp.MustCompile(`\{(.*?)\} `)
		suffix := regexp.MustCompile(` \((.*?)\)`)
		s := strings.TrimSpace(suffix.FindString(discordMember.Nick))
		s = strings.ReplaceAll(s, "(", "")
		s = strings.ReplaceAll(s, ")", "")
		m.Suffix = s
		trueNick = regRank.ReplaceAllString(discordMember.Nick, "")
		trueNick = regAlly.ReplaceAllString(trueNick, "")
		trueNick = suffix.ReplaceAllString(trueNick, "")
	}

	return trueNick
}

func (m *Member) Save() error {
	log.WithField("member", m).Debug("saving member")
	m.Updated = time.Now().UTC()

	return membersStore.Upsert(m.Id, m)
}

func (m *Member) UpdateEventCount(count int) {
	m.LegacyEvents = count
	_ = m.Save()
}

func (m *Member) IncrementEventCount() {
	m.LegacyEvents--
	_ = m.Save()
}

func (m *Member) DecrementEventCount() {
	m.LegacyEvents--
	_ = m.Save()
}

func (m *Member) IsAdmin() bool {
	logger := log.WithField("id", m.Id)
	logger.Debug("checking if admin")
	if m.Rank <= ranks.Lieutenant {
		return true
	}
	logger.Debug("is NOT admin")
	return false
}

func (m *Member) Login(code string) error {
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

		errorMessage, _ := io.ReadAll(resp.Body)
		return errors.New(string(errorMessage))
	}

	discordUser := &discordgo.Member{}
	if err := json.NewDecoder(resp.Body).Decode(&discordUser); err != nil {
		return err
	}

	if err := membersStore.Get(discordUser.User.ID).Decode(&m); err != nil {
		return errors.Wrap(err, "getting stored member")
	}

	m.Avatar = discordUser.Avatar
	_ = m.Save()

	return nil
}

func (m *Member) Delete() error {
	log.WithField("member", m).Debug("deleting member")

	return membersStore.Delete(m.Id)
}

func (m *Member) ToMap() map[string]interface{} {
	r := map[string]interface{}{}
	b, _ := json.Marshal(m)
	_ = json.Unmarshal(b, &r)
	return r
}

func (m *Member) GiveMerit(reason string, who *Member) error {
	m.Merits = append(m.Merits, &Merit{
		Giver:  *who,
		Reason: reason,
		When:   time.Now().UTC(),
	})

	return m.Save()
}

func (m *Member) GiveDemerit(reason string, who *Member) error {
	m.Demerits = append(m.Demerits, &Demerit{
		Giver:  *who,
		Reason: reason,
		When:   time.Now().UTC(),
	})

	return m.Save()
}

func (m *Member) GetOnboardingMessage() *discordgo.Message {
	onboarded := "No"
	if m.OnboardedAt != nil {
		onboarded = m.OnboardedAt.Format("2006-01-02 15:04:05 -0700 MST")
	}

	fields := []*discordgo.MessageEmbedField{
		{
			Name:  "Onboarded",
			Value: onboarded,
		},
	}

	if m.OnboardedAt != nil {
		fields = append(fields, []*discordgo.MessageEmbedField{
			{
				Name:  "Age",
				Value: "Not Set",
			},
			{
				Name:  "RSI Profile",
				Value: "None",
			},
			{
				Name:  "Primary set Org",
				Value: "None",
			},
			{
				Name:  "Affiliated Orgs",
				Value: "None",
			},
			{
				Name:  "How they found us",
				Value: "A way",
			},
			{
				Name:  "Who Recruited",
				Value: "No one",
			},
			{
				Name:  "Time Playing Star Citizen",
				Value: "Many Years",
			},
			{
				Name:  "Interested Gamplay",
				Value: "Salvage, Bounty Hunting, Mining",
			},
		}...)
	}

	description := ""
	if m.LeftAt != nil {
		description = "Left " + m.LeftAt.Format("2006-01-02 15:04:05 -0700 MST")
	}

	message := &discordgo.Message{
		Content: "<@" + m.Id + ">",
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       "Info",
				Description: description,
				Fields:      fields,
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Joined " + m.Joined.Format("2006-01-02 15:04:05 -0700 MST"),
				},
			},
		},
	}

	return message
}

func (m *Member) IsRanked() bool {
	return m.Rank <= ranks.Member
}
