package dashboard

import (
	"context"
	"fmt"
	"time"

	"github.com/sol-armada/sol-bot/settings"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Metrics represents the dashboard metrics
type Metrics struct {
	Timestamp  time.Time    `json:"timestamp"`
	Health     Health       `json:"health"`
	Members    Members      `json:"members"`
	Attendance Attendance   `json:"attendance"`
	Tokens     Tokens       `json:"tokens"`
	Activity   Activity     `json:"activity"`
	Raffles    Raffles      `json:"raffles"`
	Giveaways  Giveaways    `json:"giveaways"`
	Configs    []ConfigItem `json:"configs"`
}

type Health struct {
	DatabaseConnected bool      `json:"database_connected"`
	Uptime            string    `json:"uptime"`
	LastUpdate        time.Time `json:"last_update"`
}

type Members struct {
	Total             int            `json:"total"`
	ByRank            map[string]int `json:"by_rank"`
	Validated         int            `json:"validated"`
	Unvalidated       int            `json:"unvalidated"`
	RecentlyJoined    int            `json:"recently_joined"` // Last 7 days
	RecentlyLeft      int            `json:"recently_left"`   // Last 7 days
	RSIMembers        int            `json:"rsi_members"`
	BadAffiliations   int            `json:"bad_affiliations"`
	PendingOnboarding int            `json:"pending_onboarding"`
}

type Attendance struct {
	Active           int                `json:"active"`
	Recent           []AttendanceRecord `json:"recent"`
	TotalEvents      int                `json:"total_events"`
	SuccessfulEvents int                `json:"successful_events"`
	SuccessRate      float64            `json:"success_rate"`
	AverageAttendees float64            `json:"average_attendees"`
}

type AttendanceRecord struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	MemberCount int       `json:"member_count"`
	Successful  bool      `json:"successful"`
	DateCreated time.Time `json:"date_created"`
}

type Tokens struct {
	TotalInCirculation   int                 `json:"total_in_circulation"`
	RecentDistributions  []TokenDistribution `json:"recent_distributions"`
	TopHolders           []TokenHolder       `json:"top_holders"`
	DistributionByReason map[string]int      `json:"distribution_by_reason"`
}

type TokenDistribution struct {
	MemberName string    `json:"member_name"`
	Amount     int       `json:"amount"`
	Reason     string    `json:"reason"`
	CreatedAt  time.Time `json:"created_at"`
}

type TokenHolder struct {
	MemberID   string `json:"member_id"`
	MemberName string `json:"member_name"`
	Balance    int    `json:"balance"`
}

type Activity struct {
	ActiveToday        int `json:"active_today"`
	ActiveThisWeek     int `json:"active_this_week"`
	VoiceActivity      int `json:"voice_activity"`       // Last 24h
	MessageActivity    int `json:"message_activity"`     // Last 24h
	StarCitizenPlaying int `json:"star_citizen_playing"` // Current
}

type Raffles struct {
	Active int          `json:"active"`
	Recent []RaffleInfo `json:"recent"`
}

type RaffleInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Prize     string    `json:"prize"`
	Entries   int       `json:"entries"`
	Winners   []string  `json:"winners"`
	CreatedAt time.Time `json:"created_at"`
}

type Giveaways struct {
	Active   int            `json:"active"`
	Recent   []GiveawayInfo `json:"recent"`
	Upcoming []GiveawayInfo `json:"upcoming"`
}

type GiveawayInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ItemCount int       `json:"item_count"`
	EndTime   time.Time `json:"end_time"`
	Ended     bool      `json:"ended"`
}

type ConfigItem struct {
	Key          string    `json:"key"`
	Value        string    `json:"value"`
	DefaultValue string    `json:"default_value"`
	Type         string    `json:"type"`
	IsOverridden bool      `json:"is_overridden"`
	UpdatedBy    string    `json:"updated_by,omitempty"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// collectMetrics gathers all dashboard metrics
func (d *Dashboard) collectMetrics(ctx context.Context) (*Metrics, error) {
	metrics := &Metrics{
		Timestamp: time.Now(),
	}

	// Collect health metrics
	metrics.Health = d.collectHealthMetrics(ctx)

	// Collect member metrics
	memberMetrics, err := d.collectMemberMetrics(ctx)
	if err != nil {
		d.logger.Error("failed to collect member metrics", "error", err)
	} else {
		metrics.Members = memberMetrics
	}

	// Collect attendance metrics
	attendanceMetrics, err := d.collectAttendanceMetrics(ctx)
	if err != nil {
		d.logger.Error("failed to collect attendance metrics", "error", err)
	} else {
		metrics.Attendance = attendanceMetrics
	}

	// Collect token metrics
	tokenMetrics, err := d.collectTokenMetrics(ctx)
	if err != nil {
		d.logger.Error("failed to collect token metrics", "error", err)
	} else {
		metrics.Tokens = tokenMetrics
	}

	// Collect activity metrics
	activityMetrics, err := d.collectActivityMetrics(ctx)
	if err != nil {
		d.logger.Error("failed to collect activity metrics", "error", err)
	} else {
		metrics.Activity = activityMetrics
	}

	// Collect raffle metrics
	raffleMetrics, err := d.collectRaffleMetrics(ctx)
	if err != nil {
		d.logger.Error("failed to collect raffle metrics", "error", err)
	} else {
		metrics.Raffles = raffleMetrics
	}

	// Collect giveaway metrics
	giveawayMetrics, err := d.collectGiveawayMetrics(ctx)
	if err != nil {
		d.logger.Error("failed to collect giveaway metrics", "error", err)
	} else {
		metrics.Giveaways = giveawayMetrics
	}

	// Collect config metrics
	configMetrics, err := d.collectConfigMetrics(ctx)
	if err != nil {
		d.logger.Error("failed to collect config metrics", "error", err)
	} else {
		metrics.Configs = configMetrics
	}

	return metrics, nil
}

func (d *Dashboard) collectHealthMetrics(ctx context.Context) Health {
	return Health{
		DatabaseConnected: d.stores.Connected(ctx),
		LastUpdate:        time.Now(),
		Uptime:            "N/A", // Will be calculated from service start time
	}
}

func (d *Dashboard) collectMemberMetrics(ctx context.Context) (Members, error) {
	membersStore, ok := d.stores.GetMembersStore()
	if !ok {
		return Members{}, nil
	}

	metrics := Members{
		ByRank: make(map[string]int),
	}

	// Get total count and aggregate stats
	pipeline := mongo.Pipeline{
		{{Key: "$facet", Value: bson.D{
			{Key: "total", Value: bson.A{
				bson.D{{Key: "$count", Value: "count"}},
			}},
			{Key: "by_rank", Value: bson.A{
				bson.D{{Key: "$group", Value: bson.D{
					{Key: "_id", Value: "$rank"},
					{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
				}}},
			}},
			{Key: "validated", Value: bson.A{
				bson.D{{Key: "$match", Value: bson.D{{Key: "validated", Value: true}}}},
				bson.D{{Key: "$count", Value: "count"}},
			}},
			{Key: "unvalidated", Value: bson.A{
				bson.D{{Key: "$match", Value: bson.D{{Key: "validated", Value: false}}}},
				bson.D{{Key: "$count", Value: "count"}},
			}},
			{Key: "recently_joined", Value: bson.A{
				bson.D{{Key: "$match", Value: bson.D{{Key: "joined", Value: bson.D{{Key: "$gte", Value: time.Now().AddDate(0, 0, -7)}}}}}},
				bson.D{{Key: "$count", Value: "count"}},
			}},
			{Key: "recently_left", Value: bson.A{
				bson.D{{Key: "$match", Value: bson.D{{Key: "left_at", Value: bson.D{{Key: "$gte", Value: time.Now().AddDate(0, 0, -7)}}}}}},
				bson.D{{Key: "$count", Value: "count"}},
			}},
			{Key: "rsi_members", Value: bson.A{
				bson.D{{Key: "$match", Value: bson.D{{Key: "rsi_member", Value: true}}}},
				bson.D{{Key: "$count", Value: "count"}},
			}},
			{Key: "bad_affiliations", Value: bson.A{
				bson.D{{Key: "$match", Value: bson.D{{Key: "bad_affiliation", Value: true}}}},
				bson.D{{Key: "$count", Value: "count"}},
			}},
			{Key: "pending_onboarding", Value: bson.A{
				bson.D{{Key: "$match", Value: bson.D{{Key: "onboarded_at", Value: nil}}}},
				bson.D{{Key: "$count", Value: "count"}},
			}},
		}}},
	}

	cur, err := membersStore.Aggregate(ctx, pipeline)
	if err != nil {
		return metrics, err
	}
	defer cur.Close(ctx)

	if cur.Next(ctx) {
		var result struct {
			Total  []struct{ Count int } `bson:"total"`
			ByRank []struct {
				ID    int
				Count int
			} `bson:"by_rank"`
			Validated         []struct{ Count int } `bson:"validated"`
			Unvalidated       []struct{ Count int } `bson:"unvalidated"`
			RecentlyJoined    []struct{ Count int } `bson:"recently_joined"`
			RecentlyLeft      []struct{ Count int } `bson:"recently_left"`
			RSIMembers        []struct{ Count int } `bson:"rsi_members"`
			BadAffiliations   []struct{ Count int } `bson:"bad_affiliations"`
			PendingOnboarding []struct{ Count int } `bson:"pending_onboarding"`
		}

		if err := cur.Decode(&result); err != nil {
			return metrics, err
		}

		if len(result.Total) > 0 {
			metrics.Total = result.Total[0].Count
		}

		for _, rank := range result.ByRank {
			rankName := getRankName(rank.ID)
			metrics.ByRank[rankName] = rank.Count
		}

		if len(result.Validated) > 0 {
			metrics.Validated = result.Validated[0].Count
		}
		if len(result.Unvalidated) > 0 {
			metrics.Unvalidated = result.Unvalidated[0].Count
		}
		if len(result.RecentlyJoined) > 0 {
			metrics.RecentlyJoined = result.RecentlyJoined[0].Count
		}
		if len(result.RecentlyLeft) > 0 {
			metrics.RecentlyLeft = result.RecentlyLeft[0].Count
		}
		if len(result.RSIMembers) > 0 {
			metrics.RSIMembers = result.RSIMembers[0].Count
		}
		if len(result.BadAffiliations) > 0 {
			metrics.BadAffiliations = result.BadAffiliations[0].Count
		}
		if len(result.PendingOnboarding) > 0 {
			metrics.PendingOnboarding = result.PendingOnboarding[0].Count
		}
	}

	return metrics, nil
}

func (d *Dashboard) collectAttendanceMetrics(ctx context.Context) (Attendance, error) {
	attendanceStore, ok := d.stores.GetAttendanceStore()
	if !ok {
		return Attendance{}, nil
	}

	metrics := Attendance{
		Recent: make([]AttendanceRecord, 0),
	}

	// Count active attendance
	activeCount, err := attendanceStore.CountDocuments(ctx, bson.D{{Key: "active", Value: true}})
	if err == nil {
		metrics.Active = int(activeCount)
	}

	// Get recent attendance records
	pipeline := mongo.Pipeline{
		{{Key: "$sort", Value: bson.D{{Key: "date_created", Value: -1}}}},
		{{Key: "$limit", Value: 10}},
		{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 1},
			{Key: "name", Value: 1},
			{Key: "successful", Value: 1},
			{Key: "date_created", Value: 1},
			{Key: "member_count", Value: bson.D{{Key: "$size", Value: "$members"}}},
		}}},
	}

	cur, err := attendanceStore.Aggregate(ctx, pipeline)
	if err == nil {
		defer cur.Close(ctx)

		for cur.Next(ctx) {
			var record struct {
				ID          string    `bson:"_id"`
				Name        string    `bson:"name"`
				MemberCount int       `bson:"member_count"`
				Successful  bool      `bson:"successful"`
				DateCreated time.Time `bson:"date_created"`
			}
			if err := cur.Decode(&record); err == nil {
				metrics.Recent = append(metrics.Recent, AttendanceRecord{
					ID:          record.ID,
					Name:        record.Name,
					MemberCount: record.MemberCount,
					Successful:  record.Successful,
					DateCreated: record.DateCreated,
				})
			}
		}
	}

	// Calculate success rate and average attendees
	totalCount, _ := attendanceStore.CountDocuments(ctx, bson.D{})
	successCount, _ := attendanceStore.CountDocuments(ctx, bson.D{{Key: "successful", Value: true}})

	metrics.TotalEvents = int(totalCount)
	metrics.SuccessfulEvents = int(successCount)
	if totalCount > 0 {
		metrics.SuccessRate = float64(successCount) / float64(totalCount) * 100
	}

	return metrics, nil
}

func (d *Dashboard) collectTokenMetrics(ctx context.Context) (Tokens, error) {
	tokenStore, ok := d.stores.GetTokensStore()
	if !ok {
		return Tokens{}, nil
	}

	metrics := Tokens{
		RecentDistributions:  make([]TokenDistribution, 0),
		TopHolders:           make([]TokenHolder, 0),
		DistributionByReason: make(map[string]int),
	}

	// Get recent distributions
	pipeline := mongo.Pipeline{
		{{Key: "$sort", Value: bson.D{{Key: "created_at", Value: -1}}}},
		{{Key: "$limit", Value: 10}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "members"},
			{Key: "localField", Value: "member_id"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "member"},
		}}},
		{{Key: "$unwind", Value: bson.D{{Key: "path", Value: "$member"}, {Key: "preserveNullAndEmptyArrays", Value: true}}}},
	}

	cur, err := tokenStore.Aggregate(ctx, pipeline)
	if err == nil {
		defer cur.Close(ctx)

		for cur.Next(ctx) {
			var record struct {
				Amount    int       `bson:"amount"`
				Reason    string    `bson:"reason"`
				CreatedAt time.Time `bson:"created_at"`
				Member    struct {
					Name string `bson:"name"`
				} `bson:"member"`
			}
			if err := cur.Decode(&record); err == nil {
				metrics.RecentDistributions = append(metrics.RecentDistributions, TokenDistribution{
					MemberName: record.Member.Name,
					Amount:     record.Amount,
					Reason:     record.Reason,
					CreatedAt:  record.CreatedAt,
				})
			}
		}
	}

	// Get top token holders
	holderPipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$member_id"},
			{Key: "balance", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "balance", Value: -1}}}},
		{{Key: "$limit", Value: 10}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "members"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "member"},
		}}},
		{{Key: "$unwind", Value: bson.D{{Key: "path", Value: "$member"}, {Key: "preserveNullAndEmptyArrays", Value: true}}}},
	}

	holderCur, err := tokenStore.Aggregate(ctx, holderPipeline)
	if err == nil {
		defer holderCur.Close(ctx)

		totalCirculation := 0
		for holderCur.Next(ctx) {
			var holder struct {
				ID      string `bson:"_id"`
				Balance int    `bson:"balance"`
				Member  struct {
					Name string `bson:"name"`
				} `bson:"member"`
			}
			if err := holderCur.Decode(&holder); err == nil {
				totalCirculation += holder.Balance
				metrics.TopHolders = append(metrics.TopHolders, TokenHolder{
					MemberID:   holder.ID,
					MemberName: holder.Member.Name,
					Balance:    holder.Balance,
				})
			}
		}
		metrics.TotalInCirculation = totalCirculation
	}

	return metrics, nil
}

func (d *Dashboard) collectActivityMetrics(ctx context.Context) (Activity, error) {
	activityStore, ok := d.stores.GetActivityStore()
	if !ok {
		return Activity{}, nil
	}

	metrics := Activity{}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekAgo := now.AddDate(0, 0, -7)
	dayAgo := now.Add(-24 * time.Hour)

	// Active today
	activeToday, _ := activityStore.CountDocuments(ctx, bson.D{{Key: "when", Value: bson.D{{Key: "$gte", Value: today}}}})
	metrics.ActiveToday = int(activeToday)

	// Active this week
	activeWeek, _ := activityStore.CountDocuments(ctx, bson.D{{Key: "when", Value: bson.D{{Key: "$gte", Value: weekAgo}}}})
	metrics.ActiveThisWeek = int(activeWeek)

	// Voice activity in last 24h
	voiceActivity, _ := activityStore.CountDocuments(ctx, bson.D{
		{Key: "when", Value: bson.D{{Key: "$gte", Value: dayAgo}}},
		{Key: "meta.what", Value: bson.D{{Key: "$in", Value: bson.A{"voice_join", "voice_switch"}}}},
	})
	metrics.VoiceActivity = int(voiceActivity)

	// Message activity in last 24h
	messageActivity, _ := activityStore.CountDocuments(ctx, bson.D{
		{Key: "when", Value: bson.D{{Key: "$gte", Value: dayAgo}}},
		{Key: "meta.what", Value: "message"},
	})
	metrics.MessageActivity = int(messageActivity)

	return metrics, nil
}

func (d *Dashboard) collectRaffleMetrics(ctx context.Context) (Raffles, error) {
	raffleStore, ok := d.stores.GetRafflesStore()
	if !ok {
		return Raffles{}, nil
	}

	metrics := Raffles{
		Recent: make([]RaffleInfo, 0),
	}

	// Count active raffles
	activeCount, _ := raffleStore.CountDocuments(ctx, bson.D{{Key: "ended", Value: false}})
	metrics.Active = int(activeCount)

	// Get recent raffles
	pipeline := mongo.Pipeline{
		{{Key: "$sort", Value: bson.D{{Key: "created_at", Value: -1}}}},
		{{Key: "$limit", Value: 5}},
	}

	cur, err := raffleStore.Aggregate(ctx, pipeline)
	if err == nil {
		defer cur.Close(ctx)

		for cur.Next(ctx) {
			var raffle struct {
				ID        string         `bson:"_id"`
				Name      string         `bson:"name"`
				Prize     string         `bson:"prize"`
				Tickets   map[string]int `bson:"tickets"`
				Winners   []string       `bson:"winners"`
				CreatedAt time.Time      `bson:"created_at"`
			}
			if err := cur.Decode(&raffle); err == nil {
				entryCount := 0
				for _, count := range raffle.Tickets {
					entryCount += count
				}

				metrics.Recent = append(metrics.Recent, RaffleInfo{
					ID:        raffle.ID,
					Name:      raffle.Name,
					Prize:     raffle.Prize,
					Entries:   entryCount,
					Winners:   raffle.Winners,
					CreatedAt: raffle.CreatedAt,
				})
			}
		}
	}

	return metrics, nil
}

func (d *Dashboard) collectGiveawayMetrics(ctx context.Context) (Giveaways, error) {
	giveawayStore, ok := d.stores.GetGiveawaysStore()
	if !ok {
		return Giveaways{}, nil
	}

	metrics := Giveaways{
		Recent:   make([]GiveawayInfo, 0),
		Upcoming: make([]GiveawayInfo, 0),
	}

	// Count active giveaways
	activeCount, _ := giveawayStore.CountDocuments(ctx, bson.D{{Key: "ended", Value: false}})
	metrics.Active = int(activeCount)

	// Get recent/upcoming giveaways
	pipeline := mongo.Pipeline{
		{{Key: "$sort", Value: bson.D{{Key: "end_time", Value: -1}}}},
		{{Key: "$limit", Value: 5}},
	}

	cur, err := giveawayStore.Aggregate(ctx, pipeline)
	if err == nil {
		defer cur.Close(ctx)

		for cur.Next(ctx) {
			var giveaway struct {
				ID      string         `bson:"id"`
				Name    string         `bson:"name"`
				Items   map[string]any `bson:"items"`
				EndTime time.Time      `bson:"end_time"`
				Ended   bool           `bson:"ended"`
			}
			if err := cur.Decode(&giveaway); err == nil {
				info := GiveawayInfo{
					ID:        giveaway.ID,
					Name:      giveaway.Name,
					ItemCount: len(giveaway.Items),
					EndTime:   giveaway.EndTime,
					Ended:     giveaway.Ended,
				}

				if giveaway.Ended {
					metrics.Recent = append(metrics.Recent, info)
				} else {
					metrics.Upcoming = append(metrics.Upcoming, info)
				}
			}
		}
	}

	return metrics, nil
}

func getRankName(rank int) string {
	ranks := map[int]string{
		1: "Admiral",
		2: "Commander",
		3: "Lieutenant",
		4: "Specialist",
		5: "Technician",
		6: "Member",
		7: "Recruit",
		8: "Guest",
	}
	if name, ok := ranks[rank]; ok {
		return name
	}
	return "Unknown"
}

func (d *Dashboard) collectConfigMetrics(ctx context.Context) ([]ConfigItem, error) {
	// Define all config keys we want to expose in the dashboard
	configKeys := []struct {
		Key  string
		Type string
	}{
		// Log configs
		{"log.debug", "boolean"},
		{"log.cli", "boolean"},
		{"log.file", "string"},

		// Feature configs
		{"features.monitor.enable", "boolean"},
		{"features.merit.enable", "boolean"},
		{"features.attendance.enable", "boolean"},
		{"features.onboarding.enable", "boolean"},
		{"features.activity_tracking.enable", "boolean"},
		{"features.tokens.enable", "boolean"},
		{"features.raffles.enable", "boolean"},
		{"features.giveaways.enable", "boolean"},

		// Discord configs (non-sensitive)
		{"discord.guild_id", "string"},
		{"discord.error_channel_id", "string"},
		{"discord.channels.event_singup", "string"},

		// Attendance configs
		{"features.attendance.channel_id", "string"},
		{"features.attendance.allowed_roles", "array"},

		// Onboarding configs
		{"features.onboarding.input_channel_id", "string"},
		{"features.onboarding.output_channel_id", "string"},

		// Activity tracking configs
		{"features.activity_tracking.afk_channel_id", "string"},

		// Dashboard config
		{"dashboard.port", "string"},

		// Mongo configs (non-sensitive)
		{"mongo.host", "string"},
		{"mongo.port", "string"},
		{"mongo.database", "string"},

		// Org configs
		{"rsi_org_sid", "string"},
		{"allies", "array"},
		{"enemies", "array"},
		{"ally_role", "string"},
	}

	// Get all configs from MongoDB
	configStore, ok := d.stores.GetConfigsStore()
	if !ok {
		return []ConfigItem{}, fmt.Errorf("config store not available")
	}

	cursor, err := configStore.GetAll()
	if err != nil {
		return []ConfigItem{}, err
	}
	defer cursor.Close(ctx)

	// Build map of MongoDB configs
	mongoConfigs := make(map[string]bson.M)
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err == nil {
			if name, ok := doc["name"].(string); ok {
				mongoConfigs[name] = doc
			}
		}
	}

	// Build config items
	configItems := make([]ConfigItem, 0, len(configKeys))
	for _, ck := range configKeys {
		item := ConfigItem{
			Key:  ck.Key,
			Type: ck.Type,
		}

		// Get current value from settings (which checks MongoDB first)
		switch ck.Type {
		case "boolean":
			item.Value = fmt.Sprintf("%v", settings.GetBool(ck.Key))
		case "array":
			slice := settings.GetConfigSlice(ck.Key)
			item.Value = fmt.Sprintf("%v", slice)
		default: // string, int
			item.Value = settings.GetStringWithDefault(ck.Key, "")
		}

		// Check if overridden in MongoDB
		if mongoDoc, ok := mongoConfigs[ck.Key]; ok {
			item.IsOverridden = true
			if updatedBy, ok := mongoDoc["updated_by"].(string); ok {
				item.UpdatedBy = updatedBy
			}
			if updatedAt, ok := mongoDoc["updated_at"].(time.Time); ok {
				item.UpdatedAt = updatedAt
			}
			// Default value would be from TOML, but we don't track it here
			item.DefaultValue = "(from TOML)"
		} else {
			item.IsOverridden = false
			item.DefaultValue = item.Value
		}

		configItems = append(configItems, item)
	}

	return configItems, nil
}
