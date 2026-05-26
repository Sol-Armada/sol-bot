package members

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sol-armada/sol-bot/database/postgresql"
	"github.com/sol-armada/sol-bot/database/sqlc/dbgen"
	"github.com/sol-armada/sol-bot/ranks"
)

type memberBackend interface {
	Get(id string) (*Member, error)
	GetList(ids []string) ([]*Member, error)
	GetRandom(max int, maxRank ranks.Rank) ([]Member, error)
	List(page int) ([]Member, error)
	ListByBlueprint(blueprintID string) ([]Member, error)
	Upsert(member *Member) error
	BulkUpsert(members []Member) error
	GetIDsOnly() ([]string, error)
	Delete(id string) error
}

type postgresMembersBackend struct {
	pool    *pgxpool.Pool
	queries *dbgen.Queries
}

func (b *postgresMembersBackend) Get(id string) (*Member, error) {
	row, err := b.queries.GetMember(context.Background(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, MemberNotFound
		}
		return nil, err
	}
	member := fromPgMember(row)
	return &member, nil
}

func (b *postgresMembersBackend) GetList(ids []string) ([]*Member, error) {
	rows, err := b.queries.ListMembersByIDs(context.Background(), ids)
	if err != nil {
		return nil, err
	}
	members := make([]*Member, 0, len(rows))
	for _, row := range rows {
		member := fromPgMember(row)
		members = append(members, &member)
	}
	return members, nil
}

func (b *postgresMembersBackend) GetRandom(max int, maxRank ranks.Rank) ([]Member, error) {
	rows, err := b.queries.ListRandomMembersByRank(context.Background(), dbgen.ListRandomMembersByRankParams{
		MaxRank:   int32(maxRank),
		LimitRows: int32(max),
	})
	if err != nil {
		return nil, err
	}
	members := make([]Member, 0, len(rows))
	for _, row := range rows {
		members = append(members, fromPgMember(row))
	}
	return members, nil
}

func (b *postgresMembersBackend) List(page int) ([]Member, error) {
	offset := 0
	if page > 0 {
		offset = (page - 1) * 100
	}
	rows, err := b.queries.ListMembersPage(context.Background(), dbgen.ListMembersPageParams{
		ExcludeBots: true,
		OffsetRows:  int32(offset),
		LimitRows:   100,
	})
	if err != nil {
		return nil, err
	}
	members := make([]Member, 0, len(rows))
	for _, row := range rows {
		members = append(members, fromPgMember(row))
	}
	return members, nil
}

func (b *postgresMembersBackend) ListByBlueprint(blueprintID string) ([]Member, error) {
	rows, err := b.queries.ListMembersByBlueprint(context.Background(), blueprintID)
	if err != nil {
		return nil, err
	}
	members := make([]Member, 0, len(rows))
	for _, row := range rows {
		members = append(members, fromPgMember(row))
	}
	return members, nil
}

func (b *postgresMembersBackend) Upsert(member *Member) error {
	ctx := context.Background()
	tx, err := b.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := b.queries.WithTx(tx)
	if err := qtx.UpsertMember(ctx, dbgen.UpsertMemberParams{
		ID:          member.Id,
		Name:        member.Name,
		Rank:        int32(member.Rank),
		Joined:      toPgTimestamptz(member.Joined),
		Updated:     toPgTimestamptz(member.Updated),
		IsBot:       member.IsBot,
		IsAlly:      member.IsAlly,
		IsAffiliate: member.IsAffiliate,
		IsGuest:     member.IsGuest,
	}); err != nil {
		return err
	}
	if err := qtx.ReplaceMemberBlueprints(ctx, member.Id); err != nil {
		return err
	}
	for _, blueprintID := range member.BlueprintIds {
		if err := qtx.AddMemberBlueprint(ctx, dbgen.AddMemberBlueprintParams{
			MemberID:    member.Id,
			BlueprintID: blueprintID,
		}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (b *postgresMembersBackend) BulkUpsert(members []Member) error {
	for i := range members {
		if err := b.Upsert(&members[i]); err != nil {
			return fmt.Errorf("bulk upsert member %s: %w", members[i].Id, err)
		}
	}
	return nil
}

func (b *postgresMembersBackend) GetIDsOnly() ([]string, error) {
	return b.queries.GetMemberIDs(context.Background())
}

func (b *postgresMembersBackend) Delete(id string) error {
	return b.queries.DeleteMember(context.Background(), id)
}

func fromPgMember(row dbgen.Member) Member {
	return Member{
		Id:          row.ID,
		Name:        row.Name,
		Rank:        ranks.Rank(row.Rank),
		Joined:      fromPgTimestamptz(row.Joined),
		Updated:     fromPgTimestamptz(row.Updated),
		IsBot:       row.IsBot,
		IsAlly:      row.IsAlly,
		IsAffiliate: row.IsAffiliate,
		IsGuest:     row.IsGuest,
	}
}

func toPgTimestamptz(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t.UTC(), Valid: true}
}

func fromPgTimestamptz(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time.UTC()
}

func setupMembersBackend() error {
	pgClient := postgresql.Get()
	if pgClient == nil {
		return errors.New("postgresql client not initialized")
	}
	membersBackend = &postgresMembersBackend{pool: pgClient.Pool, queries: pgClient.Queries}
	return nil
}
