package projects

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sol-armada/sol-bot/members"
)

type TaskPriority int32

const (
	Low TaskPriority = iota
	Medium
	High
)

func (p TaskPriority) String() string {
	switch p {
	case Low:
		return "Low"
	case Medium:
		return "Medium"
	case High:
		return "High"
	default:
		return "Unknown"
	}
}

type TaskStatus struct {
	ProjectId   uuid.UUID `json:"projectId"`
	Name        string    `json:"name"`
	Position    int32     `json:"position"`
	Color       string    `json:"color"`
	IsDoneState bool      `json:"isDoneState"`
}

type Task struct {
	Id          uuid.UUID       `json:"id"`
	ProjectId   uuid.UUID       `json:"projectId"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Position    int32           `json:"position"`
	Priority    TaskPriority    `json:"priority"`
	DueAt       *time.Time      `json:"dueAt"`
	Assignee    *members.Member `json:"assignee"`
	ParentTask  *Task           `json:"parentTask"`
	Status      *TaskStatus     `json:"status"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

type TaskHistoryEntry struct {
	Id          int64           `json:"id"`
	Task        *Task           `json:"task"`
	Member      *members.Member `json:"member"`
	Action      string          `json:"action"`
	Details     map[string]any  `json:"details"`
	PerformedAt time.Time       `json:"performedAt"`
}

func NewTask(projectId uuid.UUID, title, description string, position int32, priority TaskPriority, dueAt *time.Time, assignee *members.Member, parentTask *Task, status *TaskStatus) *Task {
	return &Task{
		ProjectId:   projectId,
		Title:       title,
		Description: description,
		Position:    position,
		Priority:    priority,
		DueAt:       dueAt,
		Assignee:    assignee,
		ParentTask:  parentTask,
		Status:      status,
	}
}

func NewTaskStatus(projectId uuid.UUID, name string, position int32, color string, isDoneState bool) *TaskStatus {
	return &TaskStatus{
		ProjectId:   projectId,
		Name:        name,
		Position:    position,
		Color:       color,
		IsDoneState: isDoneState,
	}
}

func NewTaskHistoryEntry(task *Task, member *members.Member, action string, details map[string]any) *TaskHistoryEntry {
	return &TaskHistoryEntry{
		Task:        task,
		Member:      member,
		Action:      action,
		Details:     details,
		PerformedAt: time.Now().UTC(),
	}
}

func ListTaskStatuses(ctx context.Context, projectId uuid.UUID) ([]*TaskStatus, error) {
	return projectStore.ListTaskStatuses(ctx, projectId)
}

func GetTaskStatus(ctx context.Context, projectId uuid.UUID, statusName string) (*TaskStatus, error) {
	return projectStore.GetTaskStatus(ctx, projectId, statusName)
}

func CreateTaskStatus(ctx context.Context, status *TaskStatus) error {
	return projectStore.InsertTaskStatus(ctx, status)
}

func UpdateTaskStatus(ctx context.Context, status *TaskStatus) error {
	return projectStore.UpdateTaskStatus(ctx, status)
}

func DeleteTaskStatus(ctx context.Context, projectId uuid.UUID, statusName string) error {
	return projectStore.DeleteTaskStatus(ctx, projectId, statusName)
}

func ListTasks(ctx context.Context, projectId uuid.UUID) ([]*Task, error) {
	return projectStore.ListTasks(ctx, projectId)
}

func GetTask(ctx context.Context, taskId uuid.UUID) (*Task, error) {
	return projectStore.GetTask(ctx, taskId)
}

func UpdateTask(ctx context.Context, before, after *Task, member *members.Member) error {
	if err := projectStore.UpdateTask(ctx, after); err != nil {
		return err
	}

	details := map[string]any{}

	if before.Title != after.Title {
		details["title"] = map[string]string{
			"before": before.Title,
			"after":  after.Title,
		}
		details["summary"] = fmt.Sprintf("%s updated the Title from '%s' to '%s'", member.Name, before.Title, after.Title)
	}

	if before.Description != after.Description {
		details["description"] = map[string]string{
			"before": before.Description,
			"after":  after.Description,
		}
		details["summary"] = fmt.Sprintf("%s updated the Description", member.Name)
	}

	if before.Position != after.Position {
		details["position"] = map[string]int32{
			"before": before.Position,
			"after":  after.Position,
		}
		details["summary"] = fmt.Sprintf("%s updated the Position from '%d' to '%d'", member.Name, before.Position, after.Position)
	}

	if before.Priority != after.Priority {
		details["priority"] = map[string]TaskPriority{
			"before": before.Priority,
			"after":  after.Priority,
		}
		details["summary"] = fmt.Sprintf("%s updated the Priority from '%s' to '%s'", member.Name, before.Priority.String(), after.Priority.String())
	}

	if (before.DueAt == nil && after.DueAt != nil) || (before.DueAt != nil && after.DueAt == nil) || (before.DueAt != nil && after.DueAt != nil && !before.DueAt.Equal(*after.DueAt)) {
		details["dueAt"] = map[string]*time.Time{
			"before": before.DueAt,
			"after":  after.DueAt,
		}
		details["summary"] = fmt.Sprintf("%s updated the Due Date from '%v' to '%v'", member.Name, func() string {
			if before.DueAt != nil && !before.DueAt.IsZero() {
				return before.DueAt.Format(time.DateOnly)
			}
			return "No Due Date"
		}(), func() string {
			if after.DueAt != nil && !after.DueAt.IsZero() {
				return after.DueAt.Format(time.DateOnly)
			}
			return "No Due Date"
		}())
	}

	if (before.Assignee == nil && after.Assignee != nil) || (before.Assignee != nil && after.Assignee == nil) || (before.Assignee != nil && after.Assignee != nil && before.Assignee.Id != after.Assignee.Id) {
		details["assignee"] = map[string]*members.Member{
			"before": before.Assignee,
			"after":  after.Assignee,
		}
		details["summary"] = fmt.Sprintf("%s updated the Assignee from '%v' to '%v'", member.Name, func() string {
			if before.Assignee != nil {
				if before.Assignee.Id == member.Id {
					return "themselves"
				}
				return before.Assignee.Name
			}
			return "Unassigned"
		}(), func() string {
			if after.Assignee != nil {
				if after.Assignee.Id == member.Id {
					return "themselves"
				}

				return after.Assignee.Name
			}
			return "Unassigned"
		}())
	}

	if (before.Status == nil && after.Status != nil) || (before.Status != nil && after.Status == nil) || (before.Status != nil && after.Status != nil && before.Status.Name != after.Status.Name) {
		details["status"] = map[string]*TaskStatus{
			"before": before.Status,
			"after":  after.Status,
		}
		details["summary"] = fmt.Sprintf("%s updated the Status from '%v' to '%v'", member.Name, before.Status, after.Status)
	}

	if (before.ParentTask == nil && after.ParentTask != nil) || (before.ParentTask != nil && after.ParentTask == nil) || (before.ParentTask != nil && after.ParentTask != nil && before.ParentTask.Id != after.ParentTask.Id) {
		details["parentTask"] = map[string]*Task{
			"before": before.ParentTask,
			"after":  after.ParentTask,
		}
		details["summary"] = fmt.Sprintf("%s updated the Parent Task from '%v' to '%v'", member.Name, before.ParentTask, after.ParentTask)
	}

	return projectStore.InsertTaskHistoryEntry(ctx, &TaskHistoryEntry{
		Task:        after,
		Member:      member,
		Action:      "update",
		PerformedAt: time.Now().UTC(),
		Details:     details,
	})
}

func DeleteTask(ctx context.Context, taskId uuid.UUID) error {
	return projectStore.DeleteTask(ctx, taskId)
}

func (s *TaskStatus) String() string {
	return s.Name
}

func (s *TaskStatus) Equals(other *TaskStatus) bool {
	if s == nil || other == nil {
		return false
	}
	return s.ProjectId == other.ProjectId && s.Name == other.Name
}

func (t *Task) Equals(other *Task) bool {
	if t == nil || other == nil {
		return false
	}
	return t.Id == other.Id && t.ProjectId == other.ProjectId
}

func (t *Task) Delete(ctx context.Context, member *members.Member) error {
	if err := projectStore.DeleteTask(ctx, t.Id); err != nil {
		return err
	}
	return projectStore.InsertTaskHistoryEntry(ctx, &TaskHistoryEntry{
		Task:        t,
		Member:      member,
		Action:      "delete",
		PerformedAt: time.Now().UTC(),
	})
}
