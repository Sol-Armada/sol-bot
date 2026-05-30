package attendance

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sol-armada/sol-bot/database/sqlc/dbgen"
	"github.com/sol-armada/sol-bot/members"
)

//go:generate go run github.com/jmattheis/goverter/cmd/goverter@v1.9.3 gen .

// goverter:converter
// goverter:output:file ./converters.gen.go
// goverter:extend PgTypeTextToString
// goverter:extend PgTypeTimestamptzToTime
// goverter:extend ConvertSubmittedByToMember
// goverter:extend ConvertMemberIdToMember
type Converter interface {
	// goverter:map ID Id
	// goverter:map ChannelID ChannelId
	// goverter:map MessageID MessageId
	FromDbAttendanceRow(row dbgen.Attendance) *Attendance
	FromDbAttendanceRows(rows []dbgen.Attendance) []*Attendance

	// goverter:map MemberID Member
	FromDbParticipantRow(row dbgen.AttendanceParticipant) *Participant
	FromDbParticipantRows(rows []dbgen.AttendanceParticipant) []*Participant
}

// Custom converters for pgtype to native types
func PgTypeTextToString(t pgtype.Text) string {
	return t.String
}

func PgTypeTimestamptzToTime(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time.UTC()
}

func ConvertSubmittedByToMember(t pgtype.Text) *members.Member {
	member, err := members.Get(t.String)
	if err != nil {
		return nil
	}
	return member
}

func ConvertMemberIdToMember(id string) *members.Member {
	member, err := members.Get(id)
	if err != nil {
		return nil
	}
	return member
}
