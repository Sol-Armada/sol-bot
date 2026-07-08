package projects

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sol-armada/sol-bot/members"
)

type Status int32

const (
	_ Status = iota
	NotStarted
	InProgress
	Completed
	OnHold
	Cancelled
)

func (s Status) String() string {
	switch s {
	case NotStarted:
		return "Not Started"
	case InProgress:
		return "In Progress"
	case Completed:
		return "Completed"
	case OnHold:
		return "On Hold"
	case Cancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

type Project struct {
	Id          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Status      Status          `json:"status"`
	Owner       *members.Member `json:"owner"`
	DueAt       *time.Time      `json:"dueAt"`
	TaskCount   int32           `json:"taskCount"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`

	Tasks []*Task `json:"tasks"`
}

func New(name, description string, owner *members.Member, dueAt *time.Time) (*Project, error) {
	newProject := &Project{
		Name:        name,
		Description: description,
		Status:      NotStarted,
		Owner:       owner,
		DueAt:       dueAt,
	}

	if err := CreateProject(context.Background(), newProject); err != nil {
		return nil, err
	}

	return newProject, nil
}

func ListProjects(ctx context.Context) ([]*Project, error) {
	projects, err := projectStore.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, project := range projects {
		tasks, err := projectStore.ListTasks(ctx, project.Id)
		if err != nil {
			return nil, err
		}
		project.Tasks = tasks
	}

	return projectStore.List(ctx)
}

func GetProject(ctx context.Context, id uuid.UUID) (*Project, error) {
	return projectStore.Get(ctx, id)
}

func CreateProject(ctx context.Context, p *Project) error {
	p.CreatedAt = time.Now().UTC()
	p.UpdatedAt = p.CreatedAt

	if err := projectStore.Insert(ctx, p); err != nil {
		return err
	}

	toDoStatus := NewTaskStatus(p.Id, "To Do", 1, "#858585", false)
	if err := projectStore.InsertTaskStatus(ctx, toDoStatus); err != nil {
		return err
	}

	inProjectStatus := NewTaskStatus(p.Id, "In Progress", 2, "#22be22", false)
	if err := projectStore.InsertTaskStatus(ctx, inProjectStatus); err != nil {
		return err
	}

	doneStatus := NewTaskStatus(p.Id, "Done", 3, "#118cdf", true)
	if err := projectStore.InsertTaskStatus(ctx, doneStatus); err != nil {
		return err
	}

	return nil
}

func DeleteProject(ctx context.Context, id uuid.UUID) error {
	return projectStore.Delete(ctx, id)
}

func (p *Project) AddTask(ctx context.Context, task *Task) error {
	taskId := p.generateId(p.Name, p.TaskCount)
	if taskId == "" {
		return fmt.Errorf("failed to generate task ID")
	}

	task.Id = taskId

	_, err := projectStore.InsertTask(ctx, task)
	if err != nil {
		return err
	}

	p.TaskCount++
	return projectStore.Update(ctx, p)
}

func (p *Project) DeleteTask(ctx context.Context, taskId string) error {
	return projectStore.DeleteTask(ctx, taskId)
}

func (p *Project) ListTasks(ctx context.Context) ([]*Task, error) {
	return projectStore.ListTasks(ctx, p.Id)
}

func (p *Project) CompletedTasks() []*Task {
	var completedTasks []*Task
	for _, task := range p.Tasks {
		if task.Status != nil && task.Status.IsDoneState {
			completedTasks = append(completedTasks, task)
		}
	}

	return completedTasks
}

func (p *Project) Members() []*members.Member {
	memberMap := make(map[string]*members.Member)
	for _, task := range p.Tasks {
		if task.Assignee != nil {
			memberMap[task.Assignee.Id] = task.Assignee
		}
	}

	var members []*members.Member
	for _, member := range memberMap {
		members = append(members, member)
	}

	return members
}

func (p *Project) generateId(projectTitle string, taskCount int32) string {
	projectNameSplit := strings.Split(projectTitle, " ")
	if len(projectNameSplit) == 0 {
		return ""
	}

	var projectNameInitials string
	for _, word := range projectNameSplit {
		if len(word) > 0 {
			projectNameInitials += strings.ToUpper(string(word[0]))
		}
	}
	if len(projectNameInitials) == 1 {
		projectNameInitials = strings.ToUpper(string(projectNameSplit[0][0]) + string(projectNameSplit[0][1]))
	}

	return projectNameInitials + "-" + fmt.Sprintf("%d", taskCount+1)
}
