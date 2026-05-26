package activity

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sol-armada/sol-bot/database/postgresql"
	"github.com/sol-armada/sol-bot/members"
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
	NameChange       ActivityType = "name_change"
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

var activityPool *pgxpool.Pool

func Setup() error {
	pg := postgresql.Get()
	if pg == nil {
		return errors.New("postgresql client not initialized")
	}
	activityPool = pg.Pool
	return nil
}

func (a *Activity) Save() error {
	if activityPool == nil {
		return errors.New("activity store not initialized")
	}

	activityMap := map[string]any{}
	j, _ := json.Marshal(a)
	_ = json.Unmarshal(j, &activityMap)

	// convert who to just id for mongo optimization
	activityMap["who"] = a.Who.Id
	// convert when to mongo datetime
	activityMap["when"] = a.When.UTC()

	_, err := activityPool.Exec(context.Background(), `
		INSERT INTO activity_logs (who_id, occurred_at, what, where_text)
		VALUES ($1, $2, $3, $4)
	`, a.Who.Id, a.When.UTC(), string(a.Meta.What), stringifyWhere(a.Meta.Where))
	return err
}

func stringifyWhere(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
