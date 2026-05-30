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
	GetCount(memberID string) (int, error)
	GetUniqueMemberCount(days int) (int, error)
	Upsert(a *Attendance) error
	Delete(id string) error
	CreateParticipant(attendanceID string, participant *Participant) error
	ListMembers(attendanceID string) ([]*members.Member, error)
	RemoveParticipant(attendanceID string, memberID string) error
	ListParticipants(attendanceID string) ([]*Participant, error)
	UpsertParticipant(attendanceID string, participant *Participant) error
}

type postgresAttendanceBackend struct {
	pool      *pgxpool.Pool
	queries   *dbgen.Queries
	converter Converter
}

func setupAttendanceBackend() error {
	pg := postgresql.Get()
	if pg == nil {
		return errors.New("postgresql client not initialized")
	}
	attendanceStore = &postgresAttendanceBackend{
		pool:      pg.Pool,
		queries:   pg.Queries,
		converter: &ConverterImpl{},
	}
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

	return b.converter.FromDbAttendanceRow(row), nil
}

func (b *postgresAttendanceBackend) List(limit int, page int) ([]*Attendance, error) {
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
	return b.converter.FromDbAttendanceRows(rows), nil
}

func (b *postgresAttendanceBackend) ListActive(limit int) ([]*Attendance, error) {
	if limit <= 0 {
		limit = 1000
	}
	rows, err := b.queries.ListActiveAttendance(context.Background(), int32(limit))
	if err != nil {
		return nil, err
	}
	return b.converter.FromDbAttendanceRows(rows), nil
}

func (b *postgresAttendanceBackend) ListRecorded(limit int) ([]*Attendance, error) {
	if limit <= 0 {
		limit = 1000
	}
	rows, err := b.queries.ListRecordedAttendance(context.Background(), int32(limit))
	if err != nil {
		return nil, err
	}
	return b.converter.FromDbAttendanceRows(rows), nil
}

func (b *postgresAttendanceBackend) CreateParticipant(attendanceID string, participant *Participant) error {
	return b.queries.UpsertAttendanceParticipant(context.Background(), dbgen.UpsertAttendanceParticipantParams{
		AttendanceID:   attendanceID,
		MemberID:       participant.Member.Id,
		JoinedAtStart:  participant.JoinedAtStart,
		StayedUntilEnd: participant.StayedUntilEnd,
	})
}

func (b *postgresAttendanceBackend) ListMembers(attendanceID string) ([]*members.Member, error) {
	rows, err := b.queries.ListAttendanceParticipants(context.Background(), attendanceID)
	if err != nil {
		return nil, err
	}
	memberIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		memberIDs = append(memberIDs, row.MemberID)
	}
	return members.GetList(memberIDs)
}

func (b *postgresAttendanceBackend) RemoveParticipant(attendanceID string, memberID string) error {
	return b.queries.DeleteAttendanceParticipant(context.Background(), dbgen.DeleteAttendanceParticipantParams{
		AttendanceID: attendanceID,
		MemberID:     memberID,
	})
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
	defer tx.Rollback(ctx) //nolint:errcheck

	qtx := b.queries.WithTx(tx)
	params := dbgen.UpsertAttendanceParams{
		ID:          a.Id,
		Name:        a.Name,
		Recorded:    a.Recorded,
		Successful:  a.Successful,
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

	return tx.Commit(ctx)
}

func (b *postgresAttendanceBackend) Delete(id string) error {
	return b.queries.DeleteAttendance(context.Background(), id)
}

func (b *postgresAttendanceBackend) ListParticipants(attendanceID string) ([]*Participant, error) {
	rows, err := b.queries.ListAttendanceParticipants(context.Background(), attendanceID)
	if err != nil {
		return nil, err
	}
	return b.converter.FromDbParticipantRows(rows), nil
}

func (b *postgresAttendanceBackend) UpsertParticipant(attendanceID string, participant *Participant) error {
	return b.queries.UpsertAttendanceParticipant(context.Background(), dbgen.UpsertAttendanceParticipantParams{
		AttendanceID:   attendanceID,
		MemberID:       participant.Member.Id,
		JoinedAtStart:  participant.JoinedAtStart,
		StayedUntilEnd: participant.StayedUntilEnd,
		IsManager:      participant.IsManager,
	})
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
