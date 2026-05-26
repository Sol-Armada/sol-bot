package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sol-armada/sol-bot/database/sqlc/dbgen"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	ctx := context.Background()

	mongoURI := flag.String("mongo-uri", envOrDefault("MONGO_URI", "mongodb://localhost:27017"), "MongoDB connection URI")
	mongoDB := flag.String("mongo-db", envOrDefault("MONGO_DATABASE", "org"), "MongoDB database name")
	pgDSN := flag.String("pg-dsn", envOrDefault("POSTGRES_DSN", ""), "PostgreSQL DSN")
	truncate := flag.Bool("truncate", false, "Truncate destination tables before backfill")
	flag.Parse()

	if strings.TrimSpace(*pgDSN) == "" {
		log.Fatal("missing postgres DSN, set --pg-dsn or POSTGRES_DSN")
	}

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(*mongoURI))
	if err != nil {
		log.Fatalf("connect mongo: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	if err := mongoClient.Ping(ctx, nil); err != nil {
		log.Fatalf("ping mongo: %v", err)
	}

	pgPool, err := pgxpool.New(ctx, *pgDSN)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pgPool.Close()

	if err := pgPool.Ping(ctx); err != nil {
		log.Fatalf("ping postgres: %v", err)
	}

	queries := dbgen.New(pgPool)
	mdb := mongoClient.Database(*mongoDB)

	if *truncate {
		if err := truncateAll(ctx, pgPool); err != nil {
			log.Fatalf("truncate destination tables: %v", err)
		}
		log.Println("destination tables truncated")
	}

	membersCount, err := backfillMembers(ctx, mdb, queries)
	if err != nil {
		log.Fatalf("backfill members: %v", err)
	}
	log.Printf("members backfilled: %d", membersCount)

	attendanceCount, err := backfillAttendance(ctx, mdb, queries)
	if err != nil {
		log.Fatalf("backfill attendance: %v", err)
	}
	log.Printf("attendance backfilled: %d", attendanceCount)

	tokensCount, err := backfillTokens(ctx, mdb, queries)
	if err != nil {
		log.Fatalf("backfill tokens: %v", err)
	}
	log.Printf("tokens backfilled: %d", tokensCount)

	log.Println("backfill completed")
}

func truncateAll(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
TRUNCATE TABLE
	attendance_participants,
	attendance_payouts,
	tokens,
	attendance,
	member_blueprints,
	members
`)
	return err
}

func backfillMembers(ctx context.Context, mdb *mongo.Database, q *dbgen.Queries) (int, error) {
	cur, err := mdb.Collection("members").Find(ctx, bson.D{})
	if err != nil {
		return 0, err
	}
	defer cur.Close(ctx)

	count := 0
	for cur.Next(ctx) {
		var doc bson.M
		if err := cur.Decode(&doc); err != nil {
			return count, err
		}

		id := asString(doc["_id"])
		if id == "" {
			continue
		}

		joined := asTime(doc["joined"])
		if joined.IsZero() {
			joined = time.Now().UTC()
		}
		updated := asTime(doc["updated"])
		if updated.IsZero() {
			updated = joined
		}

		err := q.UpsertMember(ctx, dbgen.UpsertMemberParams{
			ID:          id,
			Name:        asString(doc["name"]),
			Rank:        int32(asInt(doc["rank"])),
			Joined:      toPgTs(joined),
			Updated:     toPgTs(updated),
			IsBot:       asBool(doc["is_bot"]),
			IsAlly:      asBool(doc["is_ally"]),
			IsAffiliate: asBool(doc["is_affiliate"]),
			IsGuest:     asBool(doc["is_guest"]),
		})
		if err != nil {
			return count, fmt.Errorf("upsert member %s: %w", id, err)
		}

		if err := q.ReplaceMemberBlueprints(ctx, id); err != nil {
			return count, fmt.Errorf("replace member blueprints %s: %w", id, err)
		}
		for _, blueprintID := range asStringSlice(doc["blueprintIds"]) {
			if blueprintID == "" {
				continue
			}
			if err := q.AddMemberBlueprint(ctx, dbgen.AddMemberBlueprintParams{MemberID: id, BlueprintID: blueprintID}); err != nil {
				return count, fmt.Errorf("add member blueprint %s/%s: %w", id, blueprintID, err)
			}
		}

		count++
	}

	if err := cur.Err(); err != nil {
		return count, err
	}
	return count, nil
}

func backfillAttendance(ctx context.Context, mdb *mongo.Database, q *dbgen.Queries) (int, error) {
	cur, err := mdb.Collection("attendance").Find(ctx, bson.D{})
	if err != nil {
		return 0, err
	}
	defer cur.Close(ctx)

	count := 0
	for cur.Next(ctx) {
		var doc bson.M
		if err := cur.Decode(&doc); err != nil {
			return count, err
		}

		id := asString(doc["_id"])
		if id == "" {
			continue
		}

		dateCreated := asTime(doc["date_created"])
		if dateCreated.IsZero() {
			dateCreated = time.Now().UTC()
		}
		dateUpdated := asTime(doc["date_updated"])
		if dateUpdated.IsZero() {
			dateUpdated = dateCreated
		}

		payoutsTotal, payoutsPerMember, payoutsOrgTake, hasPayout := payoutsFromDoc(doc["payouts"])

		err := q.UpsertAttendance(ctx, dbgen.UpsertAttendanceParams{
			ID:          id,
			Name:        asString(doc["name"]),
			SubmittedBy: toPgText(memberIDField(doc["submitted_by"])),
			Recorded:    asBool(doc["recorded"]),
			Successful:  asBool(doc["successful"]),
			Active:      asBool(doc["active"]),
			Tokenable:   asBool(doc["tokenable"]),
			Status:      asString(doc["status"]),
			ChannelID:   asString(doc["channel_id"]),
			MessageID:   asString(doc["message_id"]),
			DateCreated: toPgTs(dateCreated),
			DateUpdated: toPgTs(dateUpdated),
		})
		if err != nil {
			return count, fmt.Errorf("upsert attendance %s: %w", id, err)
		}

		if hasPayout {
			err := q.UpsertAttendancePayout(ctx, dbgen.UpsertAttendancePayoutParams{
				AttendanceID: id,
				Total:        payoutsTotal,
				PerMember:    payoutsPerMember,
				OrgTake:      payoutsOrgTake,
				DateUpdated:  toPgTs(dateUpdated),
			})
			if err != nil {
				return count, fmt.Errorf("upsert attendance payout %s: %w", id, err)
			}
		}

		if err := q.ReplaceAttendanceParticipants(ctx, id); err != nil {
			return count, fmt.Errorf("replace attendance participants %s: %w", id, err)
		}

		type participantFlags struct {
			JoinedAtStart  bool
			StayedUntilEnd bool
			HasIssue       bool
		}

		participantMap := map[string]participantFlags{}
		for _, memberID := range memberIDList(doc["members"]) {
			if memberID == "" {
				continue
			}
			participantMap[memberID] = participantFlags{}
		}
		for _, memberID := range memberIDList(doc["from_start"]) {
			if memberID == "" {
				continue
			}
			flags := participantMap[memberID]
			flags.JoinedAtStart = true
			participantMap[memberID] = flags
		}
		for _, memberID := range memberIDList(doc["with_issues"]) {
			if memberID == "" {
				continue
			}
			flags := participantMap[memberID]
			flags.HasIssue = true
			participantMap[memberID] = flags
		}
		for _, memberID := range memberIDList(doc["stayed"]) {
			if memberID == "" {
				continue
			}
			flags := participantMap[memberID]
			flags.StayedUntilEnd = true
			participantMap[memberID] = flags
		}

		for memberID, flags := range participantMap {
			err := q.UpsertAttendanceParticipant(ctx, dbgen.UpsertAttendanceParticipantParams{
				AttendanceID:   id,
				MemberID:       memberID,
				JoinedAtStart:  flags.JoinedAtStart,
				StayedUntilEnd: flags.StayedUntilEnd,
				HasIssue:       flags.HasIssue,
				UpdatedAt:      toPgTs(dateUpdated),
			})
			if err != nil {
				return count, fmt.Errorf("upsert attendance participant %s/%s: %w", id, memberID, err)
			}
		}

		count++
	}

	if err := cur.Err(); err != nil {
		return count, err
	}
	return count, nil
}

func backfillTokens(ctx context.Context, mdb *mongo.Database, q *dbgen.Queries) (int, error) {
	cur, err := mdb.Collection("tokens").Find(ctx, bson.D{})
	if err != nil {
		return 0, err
	}
	defer cur.Close(ctx)

	count := 0
	for cur.Next(ctx) {
		var doc bson.M
		if err := cur.Decode(&doc); err != nil {
			return count, err
		}

		id := asString(doc["_id"])
		if id == "" {
			continue
		}

		err := q.InsertToken(ctx, dbgen.InsertTokenParams{
			ID:           id,
			MemberID:     asString(doc["member_id"]),
			Amount:       int32(asInt(doc["amount"])),
			Reason:       asString(doc["reason"]),
			AttendanceID: toPgText(asString(doc["attendance_id"])),
			Comment:      toPgText(asString(doc["comment"])),
			GiverID:      toPgText(asString(doc["giver_id"])),
			CreatedAt:    toPgTs(asTimeWithDefault(doc["created_at"], time.Now().UTC())),
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				continue
			}
			return count, fmt.Errorf("insert token %s: %w", id, err)
		}

		count++
	}

	if err := cur.Err(); err != nil {
		return count, err
	}
	return count, nil
}

func memberIDList(v any) []string {
	src := asSlice(v)
	out := make([]string, 0, len(src))
	for _, item := range src {
		id := memberIDField(item)
		if id != "" {
			out = append(out, id)
		}
	}
	return out
}

func memberIDField(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		if id, ok := t["id"]; ok {
			return asString(id)
		}
		if id, ok := t["_id"]; ok {
			return asString(id)
		}
	case bson.M:
		if id, ok := t["id"]; ok {
			return asString(id)
		}
		if id, ok := t["_id"]; ok {
			return asString(id)
		}
	}
	return ""
}

func payoutsFromDoc(v any) (int64, int64, int64, bool) {
	m := map[string]any{}
	switch t := v.(type) {
	case map[string]any:
		m = t
	case bson.M:
		m = t
	default:
		return 0, 0, 0, false
	}
	return int64(asInt(m["total"])), int64(asInt(m["per_member"])), int64(asInt(m["org_take"])), true
}

func toPgText(v string) pgtype.Text {
	if strings.TrimSpace(v) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: v, Valid: true}
}

func toPgInt8(v int64) pgtype.Int8 {
	return pgtype.Int8{Int64: v, Valid: true}
}

func toPgTs(v time.Time) pgtype.Timestamptz {
	if v.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: v.UTC(), Valid: true}
}

func asTimeWithDefault(v any, fallback time.Time) time.Time {
	t := asTime(v)
	if t.IsZero() {
		return fallback
	}
	return t
}

func asTime(v any) time.Time {
	switch t := v.(type) {
	case time.Time:
		return t.UTC()
	case *time.Time:
		if t == nil {
			return time.Time{}
		}
		return t.UTC()
	case primitive.DateTime:
		return t.Time().UTC()
	default:
		return time.Time{}
	}
}

func asBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return strings.EqualFold(t, "true")
	default:
		return false
	}
}

func asInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int32:
		return int(t)
	case int64:
		return int(t)
	case float64:
		return int(t)
	case float32:
		return int(t)
	default:
		return 0
	}
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return ""
	}
}

func asStringSlice(v any) []string {
	src := asSlice(v)
	out := make([]string, 0, len(src))
	for _, item := range src {
		s := asString(item)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func asSlice(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case primitive.A:
		return []any(t)
	case []string:
		out := make([]any, 0, len(t))
		for _, s := range t {
			out = append(out, s)
		}
		return out
	default:
		return nil
	}
}

func envOrDefault(key, fallback string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	return val
}
