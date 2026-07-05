package raffles

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sol-armada/sol-bot/database"
	"github.com/sol-armada/sol-bot/database/sqlc/dbgen"
)

type raffleBackend interface {
	Insert(ctx context.Context, raffle *Raffle) (int32, error)
	Update(ctx context.Context, raffle *Raffle) error
	Get(ctx context.Context, id int32) (*Raffle, error)
	List(ctx context.Context, limit, offset int) ([]*Raffle, error)
	Delete(ctx context.Context, id int32) error
	GetLatest(ctx context.Context) (*Raffle, error)
}

type postgresRaffleBackend struct {
	pool      *pgxpool.Pool
	queries   *dbgen.Queries
	converter Converter
}

func setupRafflesBackend() error {
	pgClient := database.Get()
	if pgClient == nil {
		return errors.New("database client not initialized")
	}
	rafflesBackend = &postgresRaffleBackend{
		pool:      pgClient.Pool,
		queries:   pgClient.Queries,
		converter: &ConverterImpl{},
	}
	return nil
}

func (p *postgresRaffleBackend) Insert(ctx context.Context, raffle *Raffle) (int32, error) {
	ticketsJSON, err := json.Marshal(raffle.Tickets)
	if err != nil {
		return 0, err
	}

	params := dbgen.InsertRaffleParams{
		Name:         raffle.Name,
		AttendanceID: pgtype.Text{String: raffle.AttendanceId, Valid: raffle.AttendanceId != ""},
		Prize:        raffle.Prize,
		Quantity:     raffle.Quantity,
		TicketsJson:  string(ticketsJSON),
		Winners:      raffle.Winners,
		Ended:        raffle.Ended,
		Test:         raffle.Test,
		ChannelID:    raffle.ChannelId,
		MessageID:    raffle.MessageId,
		CreatedAt:    pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		UpdatedAt:    pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}

	id, err := p.queries.InsertRaffle(ctx, params)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (p *postgresRaffleBackend) Update(ctx context.Context, raffle *Raffle) error {
	ticketsJSON, err := json.Marshal(raffle.Tickets)
	if err != nil {
		return err
	}

	params := dbgen.UpdateRaffleParams{
		Name:         raffle.Name,
		AttendanceID: pgtype.Text{String: raffle.AttendanceId, Valid: raffle.AttendanceId != ""},
		Prize:        raffle.Prize,
		Quantity:     raffle.Quantity,
		TicketsJson:  string(ticketsJSON),
		Winners:      raffle.Winners,
		Ended:        raffle.Ended,
		Test:         raffle.Test,
		ChannelID:    raffle.ChannelId,
		MessageID:    raffle.MessageId,
		UpdatedAt:    pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		ID:           raffle.Id,
	}

	return p.queries.UpdateRaffle(ctx, params)
}

func (p *postgresRaffleBackend) Get(ctx context.Context, id int32) (*Raffle, error) {
	row, err := p.queries.GetRaffleById(ctx, id)
	if err != nil {
		return nil, err
	}
	return p.converter.FromGetRaffleRow(row), nil
}

func (p *postgresRaffleBackend) List(ctx context.Context, limit, offset int) ([]*Raffle, error) {
	rows, err := p.queries.ListRaffles(ctx)
	if err != nil {
		return nil, err
	}
	return p.converter.FromListRafflesRow(rows), nil
}

func (p *postgresRaffleBackend) Delete(ctx context.Context, id int32) error {
	return p.queries.DeleteRaffleById(ctx, id)
}

func (p *postgresRaffleBackend) GetLatest(ctx context.Context) (*Raffle, error) {
	row, err := p.queries.GetLatestRaffle(ctx)
	if err != nil {
		return nil, err
	}
	return p.converter.FromGetRaffleRow(row), nil
}
