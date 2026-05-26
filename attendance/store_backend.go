package attendance

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sol-armada/sol-bot/database/postgresql"
	"github.com/sol-armada/sol-bot/database/sqlc/dbgen"
	"github.com/sol-armada/sol-bot/members"
)

type attendanceBackend interface {
	Get(id string) (*Attendance, error)
	List(limit int, page int) ([]*Attendance, error)
	ListActive(limit int) ([]*Attendance, error)
	ListRecorded(limit int) ([]*Attendance, error)
	GetCount(memberID string) (int, error)
	GetUniqueMemberCount(days int) (int, error)
	Upsert(a *Attendance) error
	Delete(id string) error
}

type postgresAttendanceBackend struct {
	pool    *pgxpool.Pool
	queries *dbgen.Queries
}

func setupAttendanceBackend() error {
	pg := postgresql.Get()
	if pg == nil {
		return errors.New("postgresql client not initialized")
	}
	attendanceStore = &postgresAttendanceBackend{pool: pg.Pool, queries: pg.Queries}
	return nil
}

func (b *postgresAttendanceBackend) Get(id string) (*Attendance, error) {
	row, err := b.queries.GetAttendanceByID(context.Background(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAttendanceNotFound
		}
		return nil, err
	}
	out, err := b.hydrateRows([]dbgen.Attendance{row})
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, ErrAttendanceNotFound
	}
	return out[0], nil
}

func (b *postgresAttendanceBackend) List(limit int, page int) ([]*Attendance, error) {
	if limit <= 0 {
		rows, err := b.queries.ListAllAttendance(context.Background())
		if err != nil {
			return nil, err
		}
		return b.hydrateRows(rows)
	}

	offset := 0
	if page > 0 {
		offset = (page - 1) * limit
	}

	rows, err := b.queries.ListAttendancePage(context.Background(), dbgen.ListAttendancePageParams{
		OffsetRows: int32(offset),
		LimitRows:  int32(limit),
	})
	if err != nil {
		return nil, err
	}
	return b.hydrateRows(rows)
}

func (b *postgresAttendanceBackend) ListActive(limit int) ([]*Attendance, error) {
	if limit <= 0 {
		limit = 1000
	}
	rows, err := b.queries.ListActiveAttendance(context.Background(), int32(limit))
	if err != nil {
		return nil, err
	}
	return b.hydrateRows(rows)
}

func (b *postgresAttendanceBackend) ListRecorded(limit int) ([]*Attendance, error) {
	if limit <= 0 {
		limit = 1000
	}
	rows, err := b.queries.ListRecordedAttendance(context.Background(), int32(limit))
	if err != nil {
		return nil, err
	}
	return b.hydrateRows(rows)
}

func (b *postgresAttendanceBackend) GetCount(memberID string) (int, error) {
	count, err := b.queries.CountRecordedMemberAttendanceAfterJoin(context.Background(), memberID)
	return int(count), err
}

func (b *postgresAttendanceBackend) GetUniqueMemberCount(days int) (int, error) {
	since := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
	count, err := b.queries.CountUniqueAttendanceMembersSince(context.Background(), pgtype.Timestamptz{Time: since, Valid: true})
	return int(count), err
}

func (b *postgresAttendanceBackend) Upsert(a *Attendance) error {
	ctx := context.Background()
	tx, err := b.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := b.queries.WithTx(tx)
	params := dbgen.UpsertAttendanceParams{
		ID:          a.Id,
		Name:        a.Name,
		Recorded:    a.Recorded,
		Successful:  a.Successful,
		Active:      a.Active,
		Tokenable:   a.Tokenable,
		Status:      string(a.Status),
		ChannelID:   a.ChannelId,
		MessageID:   a.MessageId,
		DateCreated: toPgTs(a.DateCreated),
		DateUpdated: toPgTs(a.DateUpdated),
	}
	if a.SubmittedBy != nil {
		params.SubmittedBy = toPgText(a.SubmittedBy.Id)
	}
	if err := qtx.UpsertAttendance(ctx, params); err != nil {
		return err
	}

	if a.Payouts == nil {
		if err := qtx.DeleteAttendancePayout(ctx, a.Id); err != nil {
			return err
		}
	} else {
		if err := qtx.UpsertAttendancePayout(ctx, dbgen.UpsertAttendancePayoutParams{
			AttendanceID: a.Id,
			Total:        a.Payouts.Total,
			PerMember:    a.Payouts.PerMember,
			OrgTake:      a.Payouts.OrgTake,
			DateUpdated:  toPgTs(a.DateUpdated),
		}); err != nil {
			return err
		}
	}

	if err := qtx.ReplaceAttendanceParticipants(ctx, a.Id); err != nil {
		return err
	}

	type participantFlags struct {
		JoinedAtStart  bool
		StayedUntilEnd bool
		HasIssue       bool
	}
	participantMap := map[string]participantFlags{}

	for _, m := range a.Members {
		if m == nil || m.Id == "" {
			continue
		}
		flags := participantMap[m.Id]
		participantMap[m.Id] = flags
	}
	for _, memberID := range a.FromStart {
		if memberID == "" {
			continue
		}
		flags := participantMap[memberID]
		flags.JoinedAtStart = true
		participantMap[memberID] = flags
	}
	for _, memberID := range a.Stayed {
		if memberID == "" {
			continue
		}
		flags := participantMap[memberID]
		flags.StayedUntilEnd = true
		participantMap[memberID] = flags
	}
	for _, m := range a.WithIssues {
		if m == nil || m.Id == "" {
			continue
		}
		flags := participantMap[m.Id]
		flags.HasIssue = true
		participantMap[m.Id] = flags
	}

	for _, p := range a.Participants {
		if p.Member == nil || p.Member.Id == "" {
			continue
		}
		flags := participantMap[p.Member.Id]
		flags.JoinedAtStart = flags.JoinedAtStart || p.JoinedAtStart
		flags.StayedUntilEnd = flags.StayedUntilEnd || p.StayedUntilEnd
		flags.HasIssue = flags.HasIssue || p.HasIssue
		participantMap[p.Member.Id] = flags
	}

	for memberID, flags := range participantMap {
		if err := qtx.UpsertAttendanceParticipant(ctx, dbgen.UpsertAttendanceParticipantParams{
			AttendanceID:   a.Id,
			MemberID:       memberID,
			JoinedAtStart:  flags.JoinedAtStart,
			StayedUntilEnd: flags.StayedUntilEnd,
			HasIssue:       flags.HasIssue,
			UpdatedAt:      toPgTs(a.DateUpdated),
		}); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (b *postgresAttendanceBackend) Delete(id string) error {
	return b.queries.DeleteAttendance(context.Background(), id)
}

func (b *postgresAttendanceBackend) hydrateRows(rows []dbgen.Attendance) ([]*Attendance, error) {
	if len(rows) == 0 {
		return []*Attendance{}, nil
	}

	memberIDs := map[string]struct{}{}
	participantsByAttendance := map[string][]dbgen.AttendanceParticipant{}

	for _, row := range rows {
		participants, err := b.queries.ListAttendanceParticipants(context.Background(), row.ID)
		if err != nil {
			return nil, err
		}
		participantsByAttendance[row.ID] = participants
		for _, participant := range participants {
			memberIDs[participant.MemberID] = struct{}{}
		}

		if row.SubmittedBy.Valid {
			memberIDs[row.SubmittedBy.String] = struct{}{}
		}
	}

	allIDs := make([]string, 0, len(memberIDs))
	for id := range memberIDs {
		allIDs = append(allIDs, id)
	}
	memberMap := map[string]*members.Member{}
	if len(allIDs) > 0 {
		list, err := members.GetList(allIDs)
		if err != nil {
			return nil, err
		}
		for _, m := range list {
			memberMap[m.Id] = m
		}
	}

	out := make([]*Attendance, 0, len(rows))
	for _, row := range rows {
		a := &Attendance{
			Id:          row.ID,
			Name:        row.Name,
			Recorded:    row.Recorded,
			Successful:  row.Successful,
			Active:      row.Active,
			Tokenable:   row.Tokenable,
			Status:      Status(row.Status),
			FromStart:   []string{},
			Stayed:      []string{},
			ChannelId:   row.ChannelID,
			MessageId:   row.MessageID,
			DateCreated: fromPgTs(row.DateCreated),
			DateUpdated: fromPgTs(row.DateUpdated),
		}
		if row.SubmittedBy.Valid {
			if m, ok := memberMap[row.SubmittedBy.String]; ok {
				a.SubmittedBy = m
			} else {
				a.SubmittedBy = &members.Member{Id: row.SubmittedBy.String}
			}
		}
		payout, err := b.queries.GetAttendancePayout(context.Background(), row.ID)
		if err == nil {
			a.Payouts = &Payouts{
				Total:     payout.Total,
				PerMember: payout.PerMember,
				OrgTake:   payout.OrgTake,
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}

		for _, participant := range participantsByAttendance[row.ID] {
			id := participant.MemberID
			var member *members.Member
			if m, ok := memberMap[id]; ok {
				member = m
			} else {
				member = &members.Member{Id: id}
			}

			a.Members = append(a.Members, member)
			a.Participants = append(a.Participants, Participant{
				Member:         member,
				JoinedAtStart:  participant.JoinedAtStart,
				StayedUntilEnd: participant.StayedUntilEnd,
				HasIssue:       participant.HasIssue,
			})

			if participant.JoinedAtStart {
				a.FromStart = append(a.FromStart, id)
			}
			if participant.StayedUntilEnd {
				a.Stayed = append(a.Stayed, id)
			}
			if participant.HasIssue {
				a.WithIssues = append(a.WithIssues, member)
			}
		}

		out = append(out, a)
	}
	return out, nil
}

func toPgText(v string) pgtype.Text {
	if v == "" {
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

func fromPgTs(v pgtype.Timestamptz) time.Time {
	if !v.Valid {
		return time.Time{}
	}
	return v.Time.UTC()
}
