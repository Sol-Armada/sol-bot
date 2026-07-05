package raffles

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sol-armada/sol-bot/database/sqlc/dbgen"
)

//go:generate go run github.com/jmattheis/goverter/cmd/goverter@v1.9.3 gen .

// goverter:converter
// goverter:output:file ./converters.gen.go
// goverter:extend PgTypeTextToString
// goverter:extend PgTypeTimestamptzToTime
type Converter interface {
	// goverter:map ID Id
	// goverter:map AttendanceID AttendanceId
	// goverter:map TicketsJson Tickets | StringToMapStringInt
	// goverter:map ChannelID ChannelId
	// goverter:map MessageID MessageId
	FromGetRaffleRow(row dbgen.Raffle) *Raffle
	FromListRafflesRow(row []dbgen.Raffle) []*Raffle
}

func PgTypeTextToString(t pgtype.Text) string {
	return t.String
}

func StringToMapStringInt(s string) map[string]int {
	m := make(map[string]int)
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return map[string]int{}
	}
	return m
}

func PgTypeTimestamptzToTime(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time.UTC()
}
