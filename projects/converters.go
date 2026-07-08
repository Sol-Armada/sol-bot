package projects

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sol-armada/sol-bot/database/sqlc/dbgen"
	"github.com/sol-armada/sol-bot/members"
)

//go:generate go run github.com/jmattheis/goverter/cmd/goverter@v1.9.3 gen .

// goverter:converter
// goverter:output:file ./converters.gen.go
// goverter:extend PgTypeTimestamptzToTime
// goverter:extend PgTypeInt4ToInt32
// goverter:extend stringToUUID
type Converter interface {
	// goverter:map ID Id
	// goverter:map StatusID Status | toStatus
	// goverter:map OwnerID Owner | toMember
	// goverter:ignore Tasks
	FromDbProjectRow(row dbgen.Project) *Project
	FromDbProjectRows(rows []dbgen.Project) []*Project

	// goverter:map ProjectID ProjectId
	FromDbTaskStatusRow(row dbgen.TaskStatus) *TaskStatus
	FromDbTaskStatusRows(rows []dbgen.TaskStatus) []*TaskStatus

	// goverter:map ID Id
	// goverter:map ProjectID ProjectId
	// goverter:map AssigneeID Assignee | toMember
	// goverter:map . Status | toTaskStatus
	// goverter:map ParentTaskID ParentTask | toTask
	FromDbTaskRow(row dbgen.Task) *Task
	FromDbTaskRows(rows []dbgen.Task) []*Task

	// goverter:map ID Id
	// goverter:map TaskID Task | toTask
	// goverter:map PerformedByID Member | toMember
	// goverter:map TaskID Task | toTask
	// goverter:map Details Details | toMapStringAny
	FromDbTaskHistoryEntryRow(row dbgen.TaskHistory) *TaskHistoryEntry
	FromDbTaskHistoryEntryRows(rows []dbgen.TaskHistory) []*TaskHistoryEntry
}

func toStatus(statusId int32) Status {
	return Status(statusId)
}

func toTaskStatus(row dbgen.Task) *TaskStatus {
	taskUuid, err := uuid.Parse(row.ProjectID)
	if err != nil {
		return nil
	}
	status, err := projectStore.GetTaskStatus(context.Background(), taskUuid, row.StatusName)
	if err != nil {
		return nil
	}
	return status
}

func toTask[T pgtype.Text | string](taskId T) *Task {
	var idString string

	switch v := any(taskId).(type) {
	case pgtype.Text:
		if !v.Valid {
			return nil
		}
		idString = v.String
	case string:
		if v == "" {
			return nil
		}
		idString = v
	default:
		return nil
	}

	task, err := projectStore.GetTask(context.Background(), idString)
	if err != nil {
		return nil
	}
	return task
}

func toMember(memberId pgtype.Text) *members.Member {
	if !memberId.Valid {
		return nil
	}
	member, err := members.Get(memberId.String)
	if err != nil {
		return nil
	}
	return member
}

func toMapStringAny(jsonb []byte) map[string]any {
	var result map[string]any
	err := json.Unmarshal(jsonb, &result)
	if err != nil {
		return nil
	}
	return result
}

func PgTypeTimestamptzToTime(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time.UTC()
}

func stringToUUID(s string) uuid.UUID {
	u, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return u
}

func PgTypeInt4ToInt32(i pgtype.Int4) int32 {
	if !i.Valid {
		return 0
	}
	return i.Int32
}
