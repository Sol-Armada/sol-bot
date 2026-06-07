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
	Delete(id, reason string) error
	ListPromotions() ([]dbgen.ListPromotionsRow, error)
	GetRsiInfo(id string) (*RsiInfo, error)
	ListRsiInfoByHandles(handles []string) ([]*RsiInfo, error)
	UpsertRsiInfo(rsiInfo *RsiInfo) error
}

type postgresMembersBackend struct {
	pool      *pgxpool.Pool
	queries   *dbgen.Queries
	converter Converter
}

func (b *postgresMembersBackend) Get(id string) (*Member, error) {
	row, err := b.queries.GetMember(context.Background(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, MemberNotFound
		}
		return nil, err
	}
	member := b.converter.FromGetMemberRow(row)
	return &member, nil
}

func (b *postgresMembersBackend) GetList(ids []string) ([]*Member, error) {
	rows, err := b.queries.ListMembersByIDs(context.Background(), ids)
	if err != nil {
		return nil, err
	}
	convertedMembers := b.converter.FromListMembersByIDsRows(rows)
	members := make([]*Member, 0, len(convertedMembers))
	for i := range convertedMembers {
		members = append(members, &convertedMembers[i])
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
	return b.converter.FromListRandomMembersByRankRows(rows), nil
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
	return b.converter.FromListMembersPageRows(rows), nil
}

func (b *postgresMembersBackend) ListAll() ([]Member, error) {
	rows, err := b.queries.ListMembers(context.Background())
	if err != nil {
		return nil, err
	}
	return b.converter.FromListMembersRows(rows), nil
}

func (b *postgresMembersBackend) ListByBlueprint(blueprintID string) ([]Member, error) {
	rows, err := b.queries.ListMembersByBlueprint(context.Background(), blueprintID)
	if err != nil {
		return nil, err
	}
	return b.converter.FromListMembersByBlueprintRows(rows), nil
}

func (b *postgresMembersBackend) Upsert(member *Member) error {
	ctx := context.Background()
	tx, err := b.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	qtx := b.queries.WithTx(tx)

	upsertParams := dbgen.UpsertMemberParams{
		ID:          member.Id,
		Name:        member.Name,
		Rank:        int32(member.Rank),
		Joined:      toPgTimestamptz(member.Joined),
		Updated:     toPgTimestamptz(member.Updated),
		IsBot:       member.IsBot,
		IsAlly:      member.IsAlly,
		IsAffiliate: member.IsAffiliate,
		DmOptOut:    member.DmOptOut,
	}

	if member.ValidatedAt != nil && !member.ValidatedAt.IsZero() {
		upsertParams.ValidatedAt = toPgTimestamptz(*member.ValidatedAt)
	}

	if err := qtx.UpsertMember(ctx, upsertParams); err != nil {
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

func (b *postgresMembersBackend) Delete(id, reason string) error {
	return b.queries.DeleteMember(context.Background(), dbgen.DeleteMemberParams{
		ID:         id,
		DateLeft:   toPgTimestamptz(time.Now()),
		ReasonLeft: pgtype.Text{String: reason, Valid: reason != ""},
	})
}

func (b *postgresMembersBackend) ListPromotions() ([]dbgen.ListPromotionsRow, error) {
	rows, err := b.queries.ListPromotions(context.Background())
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (b *postgresMembersBackend) GetRsiInfo(handle string) (*RsiInfo, error) {
	row, err := b.queries.GetRsiInfoByHandle(context.Background(), handle)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return b.converter.FromRsiInfoRow(row), nil
}

func (b *postgresMembersBackend) ListRsiInfoByHandles(handles []string) ([]*RsiInfo, error) {
	rows, err := b.queries.ListRsiInfoByHandles(context.Background(), handles)
	if err != nil {
		return nil, err
	}
	return b.converter.FromRsiInfoRows(rows), nil
}

func (b *postgresMembersBackend) UpsertRsiInfo(rsiInfo *RsiInfo) error {
	return b.queries.UpsertRsiInfo(context.Background(), dbgen.UpsertRsiInfoParams{
		Handle:        rsiInfo.Handle,
		PrimaryOrg:    pgtype.Text{String: rsiInfo.PrimaryOrg, Valid: rsiInfo.PrimaryOrg != ""},
		PrimaryOrgSid: pgtype.Text{String: rsiInfo.PrimaryOrgSid, Valid: rsiInfo.PrimaryOrgSid != ""},
		Affiliations:  rsiInfo.Affiliations,
	})
}

func toPgTimestamptz(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t.UTC(), Valid: true}
}

func setupMembersBackend() error {
	pgClient := postgresql.Get()
	if pgClient == nil {
		return errors.New("postgresql client not initialized")
	}
	membersBackend = &postgresMembersBackend{
		pool:      pgClient.Pool,
		queries:   pgClient.Queries,
		converter: &ConverterImpl{},
	}
	return nil
}
