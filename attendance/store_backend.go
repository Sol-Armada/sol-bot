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
		FromStart:   a.FromStart,
		Stayed:      a.Stayed,
		ChannelID:   a.ChannelId,
		MessageID:   a.MessageId,
		DateCreated: toPgTs(a.DateCreated),
		DateUpdated: toPgTs(a.DateUpdated),
	}
	if a.SubmittedBy != nil {
		params.SubmittedBy = toPgText(a.SubmittedBy.Id)
	}
	if a.Payouts != nil {
		params.PayoutsTotal = toPgInt8(a.Payouts.Total)
		params.PayoutsPerMember = toPgInt8(a.Payouts.PerMember)
		params.PayoutsOrgTake = toPgInt8(a.Payouts.OrgTake)
	}

	if err := qtx.UpsertAttendance(ctx, params); err != nil {
		return err
	}
	if err := qtx.ReplaceAttendanceMembers(ctx, a.Id); err != nil {
		return err
	}
	for _, m := range a.Members {
		if m == nil || m.Id == "" {
			continue
		}
		if err := qtx.AddAttendanceMember(ctx, dbgen.AddAttendanceMemberParams{AttendanceID: a.Id, MemberID: m.Id}); err != nil {
			return err
		}
	}
	if err := qtx.ReplaceAttendanceIssues(ctx, a.Id); err != nil {
		return err
	}
	for _, m := range a.WithIssues {
		if m == nil || m.Id == "" {
			continue
		}
		if err := qtx.AddAttendanceIssue(ctx, dbgen.AddAttendanceIssueParams{AttendanceID: a.Id, MemberID: m.Id}); err != nil {
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
	membersByAttendance := map[string][]string{}
	issuesByAttendance := map[string][]string{}

	for _, row := range rows {
		ids, err := b.queries.ListAttendanceMemberIDs(context.Background(), row.ID)
		if err != nil {
			return nil, err
		}
		membersByAttendance[row.ID] = ids
		for _, id := range ids {
			memberIDs[id] = struct{}{}
		}

		issues, err := b.queries.ListAttendanceIssueIDs(context.Background(), row.ID)
		if err != nil {
			return nil, err
		}
		issuesByAttendance[row.ID] = issues
		for _, id := range issues {
			memberIDs[id] = struct{}{}
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
			FromStart:   row.FromStart,
			Stayed:      row.Stayed,
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
		if row.PayoutsTotal.Valid || row.PayoutsPerMember.Valid || row.PayoutsOrgTake.Valid {
			a.Payouts = &Payouts{
				Total:     row.PayoutsTotal.Int64,
				PerMember: row.PayoutsPerMember.Int64,
				OrgTake:   row.PayoutsOrgTake.Int64,
			}
		}

		for _, id := range membersByAttendance[row.ID] {
			if m, ok := memberMap[id]; ok {
				a.Members = append(a.Members, m)
			} else {
				a.Members = append(a.Members, &members.Member{Id: id})
			}
		}
		for _, id := range issuesByAttendance[row.ID] {
			if m, ok := memberMap[id]; ok {
				a.WithIssues = append(a.WithIssues, m)
			} else {
				a.WithIssues = append(a.WithIssues, &members.Member{Id: id})
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

func toPgInt8(v int64) pgtype.Int8 {
	return pgtype.Int8{Int64: v, Valid: true}
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
