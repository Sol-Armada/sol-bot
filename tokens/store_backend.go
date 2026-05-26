package tokens

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sol-armada/sol-bot/database/postgresql"
	"github.com/sol-armada/sol-bot/database/sqlc/dbgen"
)

type tokenBackend interface {
	Insert(record *TokenRecord) error
	ListAll() ([]TokenRecord, error)
	ListSince(since time.Time) ([]TokenRecord, error)
	ListByAttendanceID(attendanceID string) ([]TokenRecord, error)
	ListByMemberAndAttendance(memberID, attendanceID string) ([]TokenRecord, error)
	GetBalances() (map[string]int, error)
}

type postgresTokenBackend struct {
	queries *dbgen.Queries
}

func setupTokenBackend() error {
	pg := postgresql.Get()
	if pg == nil {
		return errors.New("postgresql client not initialized")
	}
	tokenStore = &postgresTokenBackend{queries: pg.Queries}
	return nil
}

func (b *postgresTokenBackend) Insert(record *TokenRecord) error {
	params := dbgen.InsertTokenParams{
		ID:        record.Id,
		MemberID:  record.MemberId,
		Amount:    int32(record.Amount),
		Reason:    string(record.Reason),
		CreatedAt: pgtype.Timestamptz{Time: record.CreatedAt.UTC(), Valid: true},
	}
	if record.AttendanceId != nil {
		params.AttendanceID = pgtype.Text{String: *record.AttendanceId, Valid: true}
	}
	if record.Comment != nil {
		params.Comment = pgtype.Text{String: *record.Comment, Valid: true}
	}
	if record.GiverId != nil {
		params.GiverID = pgtype.Text{String: *record.GiverId, Valid: true}
	}
	return b.queries.InsertToken(context.Background(), params)
}

func (b *postgresTokenBackend) ListAll() ([]TokenRecord, error) {
	rows, err := b.queries.ListAllTokens(context.Background())
	if err != nil {
		return nil, err
	}
	return fromPgTokens(rows), nil
}

func (b *postgresTokenBackend) ListSince(since time.Time) ([]TokenRecord, error) {
	rows, err := b.queries.ListTokensSince(context.Background(), pgtype.Timestamptz{Time: since.UTC(), Valid: true})
	if err != nil {
		return nil, err
	}
	return fromPgTokens(rows), nil
}

func (b *postgresTokenBackend) ListByAttendanceID(attendanceID string) ([]TokenRecord, error) {
	rows, err := b.queries.ListTokensByAttendanceID(context.Background(), pgtype.Text{String: attendanceID, Valid: true})
	if err != nil {
		return nil, err
	}
	return fromPgTokens(rows), nil
}

func (b *postgresTokenBackend) ListByMemberAndAttendance(memberID, attendanceID string) ([]TokenRecord, error) {
	rows, err := b.queries.ListTokensByMemberAndAttendance(context.Background(), dbgen.ListTokensByMemberAndAttendanceParams{
		MemberID:     memberID,
		AttendanceID: pgtype.Text{String: attendanceID, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	return fromPgTokens(rows), nil
}

func (b *postgresTokenBackend) GetBalances() (map[string]int, error) {
	rows, err := b.queries.GetTokenBalances(context.Background())
	if err != nil {
		return nil, err
	}
	result := make(map[string]int, len(rows))
	for _, row := range rows {
		result[row.MemberID] = int(row.Balance)
	}
	return result, nil
}

func fromPgTokens(rows []dbgen.Token) []TokenRecord {
	out := make([]TokenRecord, 0, len(rows))
	for _, row := range rows {
		record := TokenRecord{
			Id:        row.ID,
			MemberId:  row.MemberID,
			Amount:    int(row.Amount),
			Reason:    Reason(row.Reason),
			CreatedAt: row.CreatedAt.Time.UTC(),
		}
		if row.AttendanceID.Valid {
			attendanceID := row.AttendanceID.String
			record.AttendanceId = &attendanceID
		}
		if row.Comment.Valid {
			comment := row.Comment.String
			record.Comment = &comment
		}
		if row.GiverID.Valid {
			giverID := row.GiverID.String
			record.GiverId = &giverID
		}
		out = append(out, record)
	}
	return out
}
