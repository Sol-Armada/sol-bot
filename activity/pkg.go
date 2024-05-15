package activity

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/stores"
)

type ActivityType string

const (
	Unknown          ActivityType = "unknown"
	VoiceJoin        ActivityType = "voice_join"
	VoiceSwitch      ActivityType = "voice_switch"
	VoiceLeave       ActivityType = "voice_leave"
	VoiceAFK         ActivityType = "voice_afk"
	Message          ActivityType = "message"
	StarCitizenStart ActivityType = "star_citizen_start"
	StarCitizenStop  ActivityType = "star_citizen_stop"
)

type Meta struct {
	What  ActivityType `json:"what"`
	Where any          `json:"where"`
}

type Activity struct {
	Who  *members.Member `json:"who"`
	When time.Time       `json:"when"`
	Meta Meta            `json:"meta"`
}

var activityStore *stores.ActivityStore

func Setup() error {
	storesClient := stores.Get()
	as, ok := storesClient.GetActivityStore()
	if !ok {
		return errors.New("activity store not found")
	}
	activityStore = as
	return nil
}

func (a *Activity) Save() error {
	if activityStore == nil {
		return errors.New("activity store not initialized")
	}

	activityMap := map[string]interface{}{}
	j, _ := json.Marshal(a)
	_ = json.Unmarshal(j, &activityMap)

	// convert who to just id for mongo optimization
	activityMap["who"] = a.Who.Id
	// convert when to mongo datetime
	activityMap["when"] = a.When.UTC()

	return activityStore.Create(activityMap)
}
