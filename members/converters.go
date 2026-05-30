package members

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sol-armada/sol-bot/database/sqlc/dbgen"
	"github.com/sol-armada/sol-bot/ranks"
)

//go:generate go run github.com/jmattheis/goverter/cmd/goverter@v1.9.3 gen .

// goverter:converter
// goverter:output:file ./converters.gen.go
// goverter:extend PgTypeTextToString
// goverter:extend PgTypeTimestamptzToTime
// goverter:extend ConvertRank
type Converter interface {
	// goverter:map ID Id
	// goverter:ignore Notes RsiInfo Avatar Validated ValidatedAt ValidationCode Suffix
	// goverter:ignore MemberSince DKP DKPSpent Merits Demerits BlueprintIds OnboardedAt Age
	// goverter:ignore Pronouns Playtime Gameplay Recruiter ChannelId MessageId LeftAt FoundBy
	// goverter:ignore TimeZone Other LegacyAge LegacyPlaytime LegacyGameplay LegacyRecruiter LegacyOther
	// goverter:ignore RsiInfo
	FromGetMemberRow(row dbgen.GetMemberRow) Member

	// goverter:map ID Id
	// goverter:ignore Notes RsiInfo Avatar Validated ValidatedAt ValidationCode Suffix
	// goverter:ignore MemberSince DKP DKPSpent Merits Demerits BlueprintIds OnboardedAt Age
	// goverter:ignore Pronouns Playtime Gameplay Recruiter ChannelId MessageId LeftAt FoundBy
	// goverter:ignore TimeZone Other LegacyAge LegacyPlaytime LegacyGameplay LegacyRecruiter LegacyOther
	FromListMembersByIDsRow(row dbgen.ListMembersByIDsRow) Member

	// goverter:map ID Id
	// goverter:ignore Notes RsiInfo Avatar Validated ValidatedAt ValidationCode Suffix
	// goverter:ignore MemberSince DKP DKPSpent Merits Demerits BlueprintIds OnboardedAt Age
	// goverter:ignore Pronouns Playtime Gameplay Recruiter ChannelId MessageId LeftAt FoundBy
	// goverter:ignore TimeZone Other LegacyAge LegacyPlaytime LegacyGameplay LegacyRecruiter LegacyOther
	FromListMembersPageRow(row dbgen.ListMembersPageRow) Member

	// goverter:map ID Id
	// goverter:ignore Notes RsiInfo Avatar Validated ValidatedAt ValidationCode Suffix
	// goverter:ignore MemberSince DKP DKPSpent Merits Demerits BlueprintIds OnboardedAt Age
	// goverter:ignore Pronouns Playtime Gameplay Recruiter ChannelId MessageId LeftAt FoundBy
	// goverter:ignore TimeZone Other LegacyAge LegacyPlaytime LegacyGameplay LegacyRecruiter LegacyOther
	FromListMembersByBlueprintRow(row dbgen.ListMembersByBlueprintRow) Member

	// goverter:map ID Id
	// goverter:ignore Notes RsiInfo Avatar Validated ValidatedAt ValidationCode Suffix
	// goverter:ignore MemberSince DKP DKPSpent Merits Demerits BlueprintIds OnboardedAt Age
	// goverter:ignore Pronouns Playtime Gameplay Recruiter ChannelId MessageId LeftAt FoundBy
	// goverter:ignore TimeZone Other LegacyAge LegacyPlaytime LegacyGameplay LegacyRecruiter LegacyOther
	FromListRandomMembersByRankRow(row dbgen.ListRandomMembersByRankRow) Member

	// Batch conversions for slices
	FromGetMemberRows(rows []dbgen.GetMemberRow) []Member
	FromListMembersPageRows(rows []dbgen.ListMembersPageRow) []Member
	FromListMembersByIDsRows(rows []dbgen.ListMembersByIDsRow) []Member
	FromListMembersByBlueprintRows(rows []dbgen.ListMembersByBlueprintRow) []Member
	FromListRandomMembersByRankRows(rows []dbgen.ListRandomMembersByRankRow) []Member

	FromRsiInfoRow(row dbgen.RsiInfo) *RsiInfo
	FromRsiInfoRows(rows []dbgen.RsiInfo) []*RsiInfo
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

// Helper to convert Rank from int32
func ConvertRank(r int32) ranks.Rank {
	return ranks.Rank(r)
}
