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
	"github.com/sol-armada/sol-bot/database/migrations"
	"github.com/sol-armada/sol-bot/database/sqlc/dbgen"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MigrationReport struct {
	membersProcessed              int
	configTagsProcessed           int
	configNamesProcessed          int
	commandsProcessed             int
	sosProcessed                  int
	kanbanProcessed               int
	blueprintsProcessed           int
	attendanceProcessed           int
	attendanceSkipped             int
	attendanceNulledSubmitted     int
	attendanceParticipantsSkipped int
	tokensProcessed               int
	tokensSkipped                 int
	tokensNulledGiver             int
	tokensNulledAttendance        int
}

func (m *MigrationReport) print() {
	fmt.Println("\n========== MIGRATION REPORT ==========")
	fmt.Printf("Members:\n")
	fmt.Printf("  Processed: %d\n\n", m.membersProcessed)

	fmt.Printf("Attendance Config:\n")
	fmt.Printf("  Tags migrated: %d\n", m.configTagsProcessed)
	fmt.Printf("  Names migrated: %d\n\n", m.configNamesProcessed)

	fmt.Printf("Other Collections:\n")
	fmt.Printf("  Commands migrated: %d\n", m.commandsProcessed)
	fmt.Printf("  SOS tickets migrated: %d\n", m.sosProcessed)
	fmt.Printf("  Kanban cards migrated: %d\n", m.kanbanProcessed)
	fmt.Printf("  Blueprint docs migrated: %d\n\n", m.blueprintsProcessed)

	fmt.Printf("Attendance:\n")
	fmt.Printf("  Processed: %d\n", m.attendanceProcessed)
	fmt.Printf("  Skipped (FK violation): %d\n", m.attendanceSkipped)
	fmt.Printf("  Nulled submitted_by field: %d\n", m.attendanceNulledSubmitted)
	fmt.Printf("  Skipped participants (missing member FK): %d\n\n", m.attendanceParticipantsSkipped)

	fmt.Printf("Tokens:\n")
	fmt.Printf("  Processed: %d\n", m.tokensProcessed)
	fmt.Printf("  Skipped (missing member_id): %d\n", m.tokensSkipped)
	if m.tokensNulledGiver > 0 {
		fmt.Printf("  Nulled giver_id field: %d\n", m.tokensNulledGiver)
	}
	if m.tokensNulledAttendance > 0 {
		fmt.Printf("  Nulled attendance_id field: %d\n", m.tokensNulledAttendance)
	}

	fmt.Println("\n========== END REPORT ==========")
}

func main() {
	ctx := context.Background()

	mongoURI := flag.String("mongo-uri", envOrDefault("MONGO_URI", "mongodb://localhost:27017"), "MongoDB connection URI")
	mongoDB := flag.String("mongo-db", envOrDefault("MONGO_DATABASE", "org"), "MongoDB database name")
	pgDSN := flag.String("pg-dsn", envOrDefault("POSTGRES_DSN", ""), "PostgreSQL DSN")
	report := flag.Bool("report", true, "Print detailed migration report (default: true)")
	applySchema := flag.Bool("apply-schema", false, "Apply initial schema (000001_init.up.sql) before backfill")
	flag.Parse()

	if strings.TrimSpace(*pgDSN) == "" {
		log.Fatal("missing postgres DSN, set --pg-dsn or POSTGRES_DSN")
	}

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(*mongoURI))
	if err != nil {
		log.Fatalf("connect mongo: %v", err)
	}
	defer mongoClient.Disconnect(ctx) //nolint:errcheck

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

	if *applySchema {
		if err := applyInitialSchema(ctx, pgPool); err != nil {
			log.Fatalf("apply schema: %v", err)
		}
		log.Println("initial schema applied")
	}

	if err := ensureRequiredSchema(ctx, pgPool); err != nil {
		log.Fatal(err)
	}

	queries := dbgen.New(pgPool)
	mdb := mongoClient.Database(*mongoDB)
	rep := &MigrationReport{}

	membersCount, err := backfillMembers(ctx, mdb, queries, rep)
	if err != nil {
		log.Fatalf("backfill members: %v", err)
	}
	log.Printf("members backfilled: %d", membersCount)

	configsCount, err := backfillConfigs(ctx, mdb, queries, rep)
	if err != nil {
		log.Fatalf("backfill attendance configs: %v", err)
	}
	log.Printf("attendance config values backfilled: %d", configsCount)

	commandsCount, err := backfillCommands(ctx, mdb, queries, rep)
	if err != nil {
		log.Fatalf("backfill commands: %v", err)
	}
	log.Printf("commands backfilled: %d", commandsCount)

	sosCount, err := backfillSOSTickets(ctx, mdb, queries, rep)
	if err != nil {
		log.Fatalf("backfill sos tickets: %v", err)
	}
	log.Printf("sos tickets backfilled: %d", sosCount)

	kanbanCount, err := backfillKanbanCards(ctx, mdb, queries, rep)
	if err != nil {
		log.Fatalf("backfill kanban cards: %v", err)
	}
	log.Printf("kanban cards backfilled: %d", kanbanCount)

	blueprintsCount, err := backfillBlueprintDocs(ctx, mdb, queries, rep)
	if err != nil {
		log.Fatalf("backfill blueprint docs: %v", err)
	}
	log.Printf("blueprint docs backfilled: %d", blueprintsCount)

	attendanceCount, err := backfillAttendance(ctx, mdb, queries, rep)
	if err != nil {
		log.Fatalf("backfill attendance: %v", err)
	}
	log.Printf("attendance backfilled: %d", attendanceCount)

	tokensCount, err := backfillTokens(ctx, mdb, queries, rep)
	if err != nil {
		log.Fatalf("backfill tokens: %v", err)
	}
	log.Printf("tokens backfilled: %d", tokensCount)

	log.Println("backfill completed")

	if *report {
		rep.print()
	}
}

func ensureRequiredSchema(ctx context.Context, pool *pgxpool.Pool) error {
	requiredTables := []string{
		"members",
		"member_blueprints",
		"attendance",
		"attendance_payouts",
		"attendance_participants",
		"attendance_tags",
		"attendance_names",
		"tokens",
		"command_logs",
		"sos_tickets",
		"kanban_cards",
		"blueprint_docs",
	}

	missing := make([]string, 0)
	for _, table := range requiredTables {
		var regclass *string
		err := pool.QueryRow(ctx, "SELECT to_regclass($1)", "public."+table).Scan(&regclass)
		if err != nil {
			return fmt.Errorf("check required table %q: %w", table, err)
		}
		if regclass == nil {
			missing = append(missing, table)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf(
			"postgres schema is missing required tables: %s\nrun migrations first, or rerun with --apply-schema",
			strings.Join(missing, ", "),
		)
	}

	return nil
}

func applyInitialSchema(ctx context.Context, pool *pgxpool.Pool) error {
	log.Println("running schema migrations")
	runner := migrations.New(pool, ctx)

	if err := runner.RevertAll("database/migrations"); err != nil {
		return fmt.Errorf("revert existing migrations: %w", err)
	}

	if err := runner.ApplyAll("database/migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	migrationsApplied, err := runner.GetAppliedMigrations()
	if err == nil {
		log.Printf("schema migrations completed, count: %d\n", len(migrationsApplied))
	}

	return nil
}

func backfillMembers(ctx context.Context, mdb *mongo.Database, q *dbgen.Queries, rep *MigrationReport) (int, error) {
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
	rep.membersProcessed = count
	return count, nil
}

func backfillConfigs(ctx context.Context, mdb *mongo.Database, q *dbgen.Queries, rep *MigrationReport) (int, error) {
	if err := q.DeleteAllAttendanceTags(ctx); err != nil {
		return 0, fmt.Errorf("delete attendance tags: %w", err)
	}
	if err := q.DeleteAllAttendanceNames(ctx); err != nil {
		return 0, fmt.Errorf("delete attendance names: %w", err)
	}

	tags, err := readMongoConfigList(ctx, mdb, "attendance_tags")
	if err != nil {
		return 0, err
	}
	names, err := readMongoConfigList(ctx, mdb, "attendance_names")
	if err != nil {
		return 0, err
	}

	tagCount := 0
	seenTags := map[string]struct{}{}
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, exists := seenTags[tag]; exists {
			continue
		}
		seenTags[tag] = struct{}{}
		if err := q.InsertAttendanceTag(ctx, tag); err != nil {
			return tagCount, fmt.Errorf("insert attendance tag %q: %w", tag, err)
		}
		tagCount++
	}

	nameCount := 0
	seenNames := map[string]struct{}{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, exists := seenNames[key]; exists {
			continue
		}
		seenNames[key] = struct{}{}
		if err := q.InsertAttendanceName(ctx, name); err != nil {
			return tagCount + nameCount, fmt.Errorf("insert attendance name %q: %w", name, err)
		}
		nameCount++
	}

	rep.configTagsProcessed = tagCount
	rep.configNamesProcessed = nameCount
	return tagCount + nameCount, nil
}

func readMongoConfigList(ctx context.Context, mdb *mongo.Database, key string) ([]string, error) {
	var doc bson.M
	err := mdb.Collection("configs").FindOne(ctx, bson.M{"name": key}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("read mongo config %s: %w", key, err)
	}

	return asStringSlice(doc["value"]), nil
}

func backfillCommands(ctx context.Context, mdb *mongo.Database, q *dbgen.Queries, rep *MigrationReport) (int, error) {
	cur, err := mdb.Collection("commands").Find(ctx, bson.D{})
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

		optionsJSON := "[]"
		if optionsValue, ok := doc["options"]; ok {
			if raw, err := bson.MarshalExtJSON(optionsValue, false, false); err == nil {
				optionsJSON = string(raw)
			}
		}

		err := q.InsertCommandLog(ctx, dbgen.InsertCommandLogParams{
			Name:            asString(doc["name"]),
			OccurredAt:      toPgTs(asTimeWithDefault(doc["when"], time.Now().UTC())),
			UserID:          asString(doc["user"]),
			InteractionType: int32(asInt(doc["type"])),
			ButtonID:        asString(doc["button_id"]),
			ErrorText:       asString(doc["error"]),
			OptionsJson:     optionsJSON,
		})
		if err != nil {
			return count, fmt.Errorf("insert command log: %w", err)
		}

		count++
	}

	if err := cur.Err(); err != nil {
		return count, err
	}
	rep.commandsProcessed = count
	return count, nil
}

func backfillSOSTickets(ctx context.Context, mdb *mongo.Database, q *dbgen.Queries, rep *MigrationReport) (int, error) {
	cur, err := mdb.Collection("sos").Find(ctx, bson.D{})
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

		id := idFromAny(doc["_id"])
		if id == "" {
			continue
		}

		payloadJSON, err := bsonDocToJSON(doc)
		if err != nil {
			return count, fmt.Errorf("marshal sos ticket %s: %w", id, err)
		}

		err = q.UpsertSOSTicket(ctx, dbgen.UpsertSOSTicketParams{
			ID:          id,
			MemberID:    asString(doc["member_id"]),
			PayloadJson: payloadJSON,
			UpdatedAt:   toPgTs(time.Now().UTC()),
		})
		if err != nil {
			return count, fmt.Errorf("upsert sos ticket %s: %w", id, err)
		}

		count++
	}

	if err := cur.Err(); err != nil {
		return count, err
	}
	rep.sosProcessed = count
	return count, nil
}

func backfillKanbanCards(ctx context.Context, mdb *mongo.Database, q *dbgen.Queries, rep *MigrationReport) (int, error) {
	cur, err := mdb.Collection("kanban").Find(ctx, bson.D{})
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

		id := idFromAny(doc["_id"])
		if id == "" {
			continue
		}

		payloadJSON, err := bsonDocToJSON(doc)
		if err != nil {
			return count, fmt.Errorf("marshal kanban card %s: %w", id, err)
		}

		err = q.UpsertKanbanCard(ctx, dbgen.UpsertKanbanCardParams{
			ID:          id,
			PayloadJson: payloadJSON,
			UpdatedAt:   toPgTs(time.Now().UTC()),
		})
		if err != nil {
			return count, fmt.Errorf("upsert kanban card %s: %w", id, err)
		}

		count++
	}

	if err := cur.Err(); err != nil {
		return count, err
	}
	rep.kanbanProcessed = count
	return count, nil
}

func backfillBlueprintDocs(ctx context.Context, mdb *mongo.Database, q *dbgen.Queries, rep *MigrationReport) (int, error) {
	cur, err := mdb.Collection("blueprint").Find(ctx, bson.D{})
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

		id := idFromAny(doc["_id"])
		if id == "" {
			continue
		}

		payloadJSON, err := bsonDocToJSON(doc)
		if err != nil {
			return count, fmt.Errorf("marshal blueprint doc %s: %w", id, err)
		}

		err = q.UpsertBlueprintDoc(ctx, dbgen.UpsertBlueprintDocParams{
			ID:          id,
			PayloadJson: payloadJSON,
			UpdatedAt:   toPgTs(time.Now().UTC()),
		})
		if err != nil {
			return count, fmt.Errorf("upsert blueprint doc %s: %w", id, err)
		}

		count++
	}

	if err := cur.Err(); err != nil {
		return count, err
	}
	rep.blueprintsProcessed = count
	return count, nil
}

func bsonDocToJSON(doc bson.M) (string, error) {
	raw, err := bson.MarshalExtJSON(doc, false, false)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func idFromAny(v any) string {
	switch t := v.(type) {
	case primitive.ObjectID:
		return t.Hex()
	case *primitive.ObjectID:
		if t == nil {
			return ""
		}
		return t.Hex()
	default:
		return asString(v)
	}
}

func backfillAttendance(ctx context.Context, mdb *mongo.Database, q *dbgen.Queries, rep *MigrationReport) (int, error) {
	cur, err := mdb.Collection("attendance").Find(ctx, bson.D{})
	if err != nil {
		return 0, err
	}
	defer cur.Close(ctx)

	count := 0
	skipped := 0
	nulledSubmitted := 0
	participantsSkipped := 0
	memberExistsCache := map[string]bool{}
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

		submittedByID := memberIDField(doc["submitted_by"])
		submittedByWasNulled := false
		if submittedByID != "" {
			if exists, ok := memberExistsCache[submittedByID]; ok {
				if !exists {
					submittedByID = ""
					submittedByWasNulled = true
				}
			} else {
				_, err := q.GetMember(ctx, submittedByID)
				exists := err == nil
				memberExistsCache[submittedByID] = exists
				if !exists {
					submittedByID = ""
					submittedByWasNulled = true
				}
			}
		}

		err := q.UpsertAttendance(ctx, dbgen.UpsertAttendanceParams{
			ID:          id,
			Name:        asString(doc["name"]),
			SubmittedBy: toPgText(submittedByID),
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

		if submittedByWasNulled {
			nulledSubmitted++
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
			if memberID == "" {
				continue
			}
			if exists, ok := memberExistsCache[memberID]; ok {
				if !exists {
					participantsSkipped++
					continue
				}
			} else {
				_, err := q.GetMember(ctx, memberID)
				exists := err == nil
				memberExistsCache[memberID] = exists
				if !exists {
					participantsSkipped++
					continue
				}
			}

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

	rep.attendanceProcessed = count
	rep.attendanceSkipped = skipped
	rep.attendanceNulledSubmitted = nulledSubmitted
	rep.attendanceParticipantsSkipped = participantsSkipped
	return count, nil
}

func backfillTokens(ctx context.Context, mdb *mongo.Database, q *dbgen.Queries, rep *MigrationReport) (int, error) {
	cur, err := mdb.Collection("tokens").Find(ctx, bson.D{})
	if err != nil {
		return 0, err
	}
	defer cur.Close(ctx)

	count := 0
	skipped := 0
	nulledGiver := 0
	nulledAttendance := 0
	memberExistsCache := map[string]bool{}
	attendanceExistsCache := map[string]bool{}
	for cur.Next(ctx) {
		var doc bson.M
		if err := cur.Decode(&doc); err != nil {
			return count, err
		}

		id := asString(doc["_id"])
		if id == "" {
			continue
		}

		memberID := asString(doc["member_id"])
		if memberID == "" {
			continue
		}
		if exists, ok := memberExistsCache[memberID]; ok {
			if !exists {
				skipped++
				continue
			}
		} else {
			_, err := q.GetMember(ctx, memberID)
			exists := err == nil
			memberExistsCache[memberID] = exists
			if !exists {
				skipped++
				continue
			}
		}

		attendanceID := asString(doc["attendance_id"])
		attendanceWasNulled := false
		if attendanceID != "" {
			if exists, ok := attendanceExistsCache[attendanceID]; ok {
				if !exists {
					attendanceID = ""
					attendanceWasNulled = true
				}
			} else {
				_, err := q.GetAttendanceByID(ctx, attendanceID)
				exists := err == nil
				attendanceExistsCache[attendanceID] = exists
				if !exists {
					attendanceID = ""
					attendanceWasNulled = true
				}
			}
		}

		giverID := asString(doc["giver_id"])
		giverWasNulled := false
		if giverID != "" {
			if exists, ok := memberExistsCache[giverID]; ok {
				if !exists {
					giverID = ""
					giverWasNulled = true
				}
			} else {
				_, err := q.GetMember(ctx, giverID)
				exists := err == nil
				memberExistsCache[giverID] = exists
				if !exists {
					giverID = ""
					giverWasNulled = true
				}
			}
		}

		err := q.InsertToken(ctx, dbgen.InsertTokenParams{
			ID:           id,
			MemberID:     memberID,
			Amount:       int32(asInt(doc["amount"])),
			Reason:       asString(doc["reason"]),
			AttendanceID: toPgText(attendanceID),
			Comment:      toPgText(asString(doc["comment"])),
			GiverID:      toPgText(giverID),
			CreatedAt:    toPgTs(asTimeWithDefault(doc["created_at"], time.Now().UTC())),
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				continue
			}
			return count, fmt.Errorf("insert token %s: %w", id, err)
		}

		if attendanceWasNulled {
			nulledAttendance++
		}
		if giverWasNulled {
			nulledGiver++
		}

		count++
	}

	if err := cur.Err(); err != nil {
		return count, err
	}

	rep.tokensProcessed = count
	rep.tokensSkipped = skipped
	rep.tokensNulledGiver = nulledGiver
	rep.tokensNulledAttendance = nulledAttendance
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

func toPgText(v string) pgtype.Text {
	if strings.TrimSpace(v) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: v, Valid: true}
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
