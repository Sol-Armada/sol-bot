package rolls

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sol-armada/sol-bot/database/postgresql"
	"github.com/sol-armada/sol-bot/database/sqlc/dbgen"
)

type rollBackend interface {
	GetEvent(id string) (*RollEvent, error)
	ListActiveEvents(limit int) ([]*RollEvent, error)
	UpsertEvent(event *RollEvent) error
	MarkEventEnded(id string, updatedAt time.Time) error
	DeleteEvent(id string) error

	UpsertItem(item *RollItem) error
	ListItemsByEvent(rollEventId string) ([]*RollItem, error)

	UpsertEntry(entry *RollEntry) error
	ListEntriesByEvent(rollEventId string) ([]*RollEntry, error)
	ListEntriesByItem(rollItemId string) ([]*RollEntry, error)
	DeleteEntry(rollItemId, memberId string) error
}

type postgresRollBackend struct {
	queries *dbgen.Queries
}

func setupRollBackend() error {
	pg := postgresql.Get()
	if pg == nil {
		return errors.New("postgresql client not initialized")
	}

	rollStore = &postgresRollBackend{queries: pg.Queries}
	return nil
}

func (b *postgresRollBackend) GetEvent(id string) (*RollEvent, error) {
	row, err := b.queries.GetRollEventByID(context.Background(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return fromDBRollEvent(row), nil
}

func (b *postgresRollBackend) ListActiveEvents(limit int) ([]*RollEvent, error) {
	rows, err := b.queries.ListActiveRollEvents(context.Background(), int32(limit))
	if err != nil {
		return nil, err
	}

	result := make([]*RollEvent, 0, len(rows))
	for _, row := range rows {
		r := fromDBRollEvent(row)
		result = append(result, r)
	}

	return result, nil
}

func (b *postgresRollBackend) UpsertEvent(event *RollEvent) error {
	if event == nil {
		return nil
	}

	return b.queries.UpsertRollEvent(context.Background(), dbgen.UpsertRollEventParams{
		ID:             event.Id,
		Name:           event.Name,
		AttendanceID:   toPgText(event.AttendanceId),
		EndTime:        toPgTs(event.EndTime),
		Ended:          event.Ended,
		ChannelID:      event.ChannelId,
		EmbedMessageID: event.EmbedMessageId,
		InputMessageID: event.InputMessageId,
		CreatedAt:      pgtype.Timestamptz{Time: event.CreatedAt.UTC(), Valid: true},
		UpdatedAt:      pgtype.Timestamptz{Time: event.UpdatedAt.UTC(), Valid: true},
	})
}

func (b *postgresRollBackend) MarkEventEnded(id string, updatedAt time.Time) error {
	return b.queries.MarkRollEventEnded(context.Background(), dbgen.MarkRollEventEndedParams{
		ID:        id,
		UpdatedAt: pgtype.Timestamptz{Time: updatedAt.UTC(), Valid: true},
	})
}

func (b *postgresRollBackend) DeleteEvent(id string) error {
	return b.queries.DeleteRollEvent(context.Background(), id)
}

func (b *postgresRollBackend) UpsertItem(item *RollItem) error {
	if item == nil {
		return nil
	}

	return b.queries.UpsertRollItem(context.Background(), dbgen.UpsertRollItemParams{
		ID:          item.Id,
		RollEventID: item.RollEventId,
		Name:        item.Name,
		Amount:      int32(item.Amount),
		SortOrder:   int32(item.SortOrder),
		CreatedAt:   pgtype.Timestamptz{Time: item.CreatedAt.UTC(), Valid: true},
	})
}

func (b *postgresRollBackend) ListItemsByEvent(rollEventId string) ([]*RollItem, error) {
	rows, err := b.queries.ListRollItemsByEvent(context.Background(), rollEventId)
	if err != nil {
		return nil, err
	}

	result := make([]*RollItem, 0, len(rows))
	for _, row := range rows {
		item := fromDBRollItem(row)
		result = append(result, item)
	}

	return result, nil
}

func (b *postgresRollBackend) UpsertEntry(entry *RollEntry) error {
	if entry == nil {
		return nil
	}

	return b.queries.UpsertRollEntry(context.Background(), dbgen.UpsertRollEntryParams{
		RollEventID: entry.RollEventId,
		RollItemID:  entry.RollItemId,
		MemberID:    entry.MemberId,
		Choice:      string(entry.Choice),
		RollValue:   toPgInt4(entry.RollValue),
		Winner:      entry.Winner,
		CreatedAt:   pgtype.Timestamptz{Time: entry.CreatedAt.UTC(), Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Time: entry.UpdatedAt.UTC(), Valid: true},
	})
}

func (b *postgresRollBackend) ListEntriesByEvent(rollEventId string) ([]*RollEntry, error) {
	rows, err := b.queries.ListRollEntriesByEvent(context.Background(), rollEventId)
	if err != nil {
		return nil, err
	}

	result := make([]*RollEntry, 0, len(rows))
	for _, row := range rows {
		entry := fromDBRollEntry(row)
		result = append(result, entry)
	}

	return result, nil
}

func (b *postgresRollBackend) ListEntriesByItem(rollItemId string) ([]*RollEntry, error) {
	rows, err := b.queries.ListRollEntriesByItem(context.Background(), rollItemId)
	if err != nil {
		return nil, err
	}

	result := make([]*RollEntry, 0, len(rows))
	for _, row := range rows {
		entry := fromDBRollEntry(row)
		result = append(result, entry)
	}

	return result, nil
}

func (b *postgresRollBackend) DeleteEntry(rollItemId, memberId string) error {
	return b.queries.DeleteRollEntry(context.Background(), dbgen.DeleteRollEntryParams{
		RollItemID: rollItemId,
		MemberID:   memberId,
	})
}

func fromDBRollEvent(row dbgen.RollEvent) *RollEvent {
	event := &RollEvent{
		Id:             row.InputMessageID,
		Name:           row.Name,
		Ended:          row.Ended,
		ChannelId:      row.ChannelID,
		EmbedMessageId: row.EmbedMessageID,
		InputMessageId: row.InputMessageID,
		CreatedAt:      row.CreatedAt.Time.UTC(),
		UpdatedAt:      row.UpdatedAt.Time.UTC(),
	}

	if row.AttendanceID.Valid {
		attendanceId := row.AttendanceID.String
		event.AttendanceId = &attendanceId
	}

	if row.EndTime.Valid {
		endTime := row.EndTime.Time.UTC()
		event.EndTime = &endTime
	}

	return event
}

func fromDBRollItem(row dbgen.RollItem) *RollItem {
	return &RollItem{
		Id:          row.ID,
		RollEventId: row.RollEventID,
		Name:        row.Name,
		Amount:      int(row.Amount),
		SortOrder:   int(row.SortOrder),
		CreatedAt:   row.CreatedAt.Time.UTC(),
	}
}

func fromDBRollEntry(row dbgen.RollEntry) *RollEntry {
	entry := &RollEntry{
		RollEventId: row.RollEventID,
		RollItemId:  row.RollItemID,
		MemberId:    row.MemberID,
		Choice:      Choice(row.Choice),
		Winner:      row.Winner,
		CreatedAt:   row.CreatedAt.Time.UTC(),
		UpdatedAt:   row.UpdatedAt.Time.UTC(),
	}

	if row.RollValue.Valid {
		rollValue := int(row.RollValue.Int32)
		entry.RollValue = &rollValue
	}

	return entry
}

func toPgText(v *string) pgtype.Text {
	if v == nil || *v == "" {
		return pgtype.Text{}
	}

	return pgtype.Text{String: *v, Valid: true}
}

func toPgTs(v *time.Time) pgtype.Timestamptz {
	if v == nil || v.IsZero() {
		return pgtype.Timestamptz{}
	}

	return pgtype.Timestamptz{Time: v.UTC(), Valid: true}
}

func toPgInt4(v *int) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{}
	}

	return pgtype.Int4{Int32: int32(*v), Valid: true}
}
