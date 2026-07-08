package projects

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sol-armada/sol-bot/database"
	"github.com/sol-armada/sol-bot/database/sqlc/dbgen"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/utils"
)

type projectBackend interface {
	List(ctx context.Context) ([]*Project, error)
	Get(ctx context.Context, id uuid.UUID) (*Project, error)
	Insert(ctx context.Context, p *Project) error
	Update(ctx context.Context, p *Project) error
	Delete(ctx context.Context, id uuid.UUID) error

	ListTasks(ctx context.Context, projectId uuid.UUID) ([]*Task, error)
	GetTask(ctx context.Context, taskId string) (*Task, error)
	InsertTask(ctx context.Context, task *Task) (*Task, error)
	UpdateTask(ctx context.Context, task *Task) error
	DeleteTask(ctx context.Context, taskId string) error

	ListTaskStatuses(ctx context.Context, projectId uuid.UUID) ([]*TaskStatus, error)
	GetTaskStatus(ctx context.Context, projectId uuid.UUID, statusName string) (*TaskStatus, error)
	InsertTaskStatus(ctx context.Context, status *TaskStatus) error
	UpdateTaskStatus(ctx context.Context, status *TaskStatus) error
	DeleteTaskStatus(ctx context.Context, projectId uuid.UUID, statusName string) error

	ListTaskHistory(ctx context.Context, taskId string) ([]*TaskHistoryEntry, error)
	InsertTaskHistoryEntry(ctx context.Context, entry *TaskHistoryEntry) error
}

type postgresProjectBackend struct {
	pool      *pgxpool.Pool
	queries   *dbgen.Queries
	converter Converter
}

func setupProjectBackend() error {
	pg := database.Get()
	if pg == nil {
		return errors.New("database client not initialized")
	}
	projectStore = &postgresProjectBackend{
		pool:      pg.Pool,
		queries:   pg.Queries,
		converter: &ConverterImpl{},
	}
	return nil
}

func (b *postgresProjectBackend) List(ctx context.Context) ([]*Project, error) {
	rows, err := b.queries.ListProjects(ctx)
	if err != nil {
		return nil, err
	}

	return b.converter.FromDbProjectRows(rows), nil
}

func (b *postgresProjectBackend) Get(ctx context.Context, id uuid.UUID) (*Project, error) {
	row, err := b.queries.GetProjectById(ctx, id.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}

	return b.converter.FromDbProjectRow(row), nil
}

func (b *postgresProjectBackend) Insert(ctx context.Context, p *Project) error {
	params := dbgen.InsertProjectParams{
		ID:          uuid.New().String(),
		Name:        p.Name,
		Description: p.Description,
		StatusID:    int32(p.Status),
		CreatedAt:   utils.ToPgTimestamptz(cmp.Or(p.CreatedAt, time.Now().UTC())),
		UpdatedAt:   utils.ToPgTimestamptz(cmp.Or(p.UpdatedAt, time.Now().UTC())),
	}

	if p.Owner != nil {
		params.OwnerID = utils.ToPgText(p.Owner.Id)
	}

	if p.DueAt != nil {
		params.DueAt = utils.ToPgTimestamptz(*p.DueAt)
	}

	newProjectId, err := b.queries.InsertProject(ctx, params)
	if err != nil {
		return err
	}

	p.Id, err = uuid.Parse(newProjectId)
	if err != nil {
		return err
	}

	return nil
}

func (b *postgresProjectBackend) Update(ctx context.Context, p *Project) error {
	params := dbgen.UpdateProjectParams{
		ID:          p.Id.String(),
		Name:        p.Name,
		Description: p.Description,
		StatusID:    int32(p.Status),
		TaskCount:   utils.ToPgInt4(p.TaskCount),
		UpdatedAt:   utils.ToPgTimestamptz(cmp.Or(p.UpdatedAt, time.Now().UTC())),
	}

	if p.Owner != nil {
		params.OwnerID = utils.ToPgText(p.Owner.Id)
	}

	if p.DueAt != nil {
		params.DueAt = utils.ToPgTimestamptz(*p.DueAt)
	}

	return b.queries.UpdateProject(ctx, params)
}

func (b *postgresProjectBackend) Delete(ctx context.Context, id uuid.UUID) error {
	return b.queries.DeleteProject(ctx, id.String())
}

func (b *postgresProjectBackend) ListTaskStatuses(ctx context.Context, projectID uuid.UUID) ([]*TaskStatus, error) {
	rows, err := b.queries.ListTaskStatuses(ctx, projectID.String())
	if err != nil {
		return nil, err
	}

	var statuses []*TaskStatus
	for _, row := range rows {
		statuses = append(statuses, &TaskStatus{
			ProjectId: projectID,
			Name:      row.Name,
			Position:  row.Position,
			Color:     row.Color,
		})
	}

	return statuses, nil
}

func (b *postgresProjectBackend) GetTaskStatus(ctx context.Context, projectID uuid.UUID, statusName string) (*TaskStatus, error) {
	row, err := b.queries.GetTaskStatus(ctx, dbgen.GetTaskStatusParams{
		ProjectID: projectID.String(),
		Name:      statusName,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTaskStatusNotFound
		}
		return nil, err
	}

	return &TaskStatus{
		ProjectId: projectID,
		Name:      row.Name,
		Position:  row.Position,
		Color:     row.Color,
	}, nil
}

func (b *postgresProjectBackend) InsertTaskStatus(ctx context.Context, status *TaskStatus) error {
	params := dbgen.InsertTaskStatusParams{
		ProjectID: status.ProjectId.String(),
		Name:      status.Name,
		Position:  status.Position,
		Color:     status.Color,
	}

	_, err := b.queries.InsertTaskStatus(ctx, params)
	return err
}

func (b *postgresProjectBackend) UpdateTaskStatus(ctx context.Context, status *TaskStatus) error {
	params := dbgen.UpdateTaskStatusParams{
		ProjectID:   status.ProjectId.String(),
		Name:        status.Name,
		Position:    status.Position,
		Color:       status.Color,
		IsDoneState: status.IsDoneState,
	}
	return b.queries.UpdateTaskStatus(ctx, params)
}

func (b *postgresProjectBackend) DeleteTaskStatus(ctx context.Context, projectID uuid.UUID, statusName string) error {
	return b.queries.DeleteTaskStatus(ctx, dbgen.DeleteTaskStatusParams{
		ProjectID: projectID.String(),
		Name:      statusName,
	})
}

func (b *postgresProjectBackend) ListTasks(ctx context.Context, projectID uuid.UUID) ([]*Task, error) {
	rows, err := b.queries.ListTasks(ctx, projectID.String())
	if err != nil {
		return nil, err
	}

	return b.converter.FromDbTaskRows(rows), nil
}

func (b *postgresProjectBackend) GetTask(ctx context.Context, taskId string) (*Task, error) {
	row, err := b.queries.GetTaskById(ctx, taskId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	return b.converter.FromDbTaskRow(row), nil
}

func (b *postgresProjectBackend) InsertTask(ctx context.Context, task *Task) (*Task, error) {
	params := dbgen.InsertTaskParams{
		ID:          task.Id,
		ProjectID:   task.ProjectId.String(),
		StatusName:  task.Status.Name,
		Title:       task.Title,
		Description: task.Description,
		Position:    task.Position,
		CreatedAt:   utils.ToPgTimestamptz(cmp.Or(task.CreatedAt, time.Now().UTC())),
		UpdatedAt:   utils.ToPgTimestamptz(cmp.Or(task.UpdatedAt, time.Now().UTC())),
	}

	if task.Assignee != nil {
		params.AssigneeID = utils.ToPgText(task.Assignee.Id)
	}

	if task.DueAt != nil {
		params.DueAt = utils.ToPgTimestamptz(*task.DueAt)
	}

	if task.ParentTask != nil {
		params.ParentTaskID = utils.ToPgText(task.ParentTask.Id)
	}

	_, err := b.queries.InsertTask(ctx, params)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (b *postgresProjectBackend) UpdateTask(ctx context.Context, task *Task) error {
	params := dbgen.UpdateTaskParams{
		ID:          task.Id,
		StatusName:  task.Status.Name,
		Title:       task.Title,
		Description: task.Description,
		Position:    task.Position,
		Priority:    int32(task.Priority),
		UpdatedAt:   utils.ToPgTimestamptz(cmp.Or(task.UpdatedAt, time.Now().UTC())),
	}

	if task.Assignee != nil {
		params.AssigneeID = utils.ToPgText(task.Assignee.Id)
	}

	if task.DueAt != nil {
		params.DueAt = utils.ToPgTimestamptz(*task.DueAt)
	}

	if task.ParentTask != nil {
		params.ParentTaskID = utils.ToPgText(task.ParentTask.Id)
	}

	return b.queries.UpdateTask(ctx, params)
}

func (b *postgresProjectBackend) DeleteTask(ctx context.Context, taskId string) error {
	return b.queries.DeleteTask(ctx, taskId)
}

func (b *postgresProjectBackend) ListTaskHistory(ctx context.Context, taskID string) ([]*TaskHistoryEntry, error) {
	rows, err := b.queries.ListTaskHistory(ctx, taskID)
	if err != nil {
		return nil, err
	}

	var history []*TaskHistoryEntry
	for _, row := range rows {
		member, err := members.Get(row.PerformedByID.String)
		if err != nil {
			return nil, err
		}

		history = append(history, &TaskHistoryEntry{
			Task:   &Task{Id: taskID},
			Member: member,
			Action: row.Action,
			// Details:   row.Details,
			// Timestamp: utils.PgTypeTimestamptzToTime(row.PerformedAt),
		})
	}

	return history, nil
}

func (b *postgresProjectBackend) InsertTaskHistoryEntry(ctx context.Context, entry *TaskHistoryEntry) error {
	jsonBytes, err := json.Marshal(entry.Details)
	if err != nil {
		return err
	}

	params := dbgen.InsertTaskHistoryParams{
		TaskID:        entry.Task.Id,
		Action:        entry.Action,
		PerformedByID: utils.ToPgText(entry.Member.Id),
		PerformedAt:   utils.ToPgTimestamptz(entry.PerformedAt),
		Details:       jsonBytes,
	}

	return b.queries.InsertTaskHistory(ctx, params)
}
