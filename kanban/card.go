package kanban

import (
	"encoding/json"
	"time"

	"github.com/sol-armada/sol-bot/members"
)

type Card struct {
	Id          string          ``
	Title       string          ``
	Description string          ``
	Status      string          ``
	Assignee    *members.Member ``
	CreatedBy   *members.Member ``
	CreatedAt   *time.Time      ``
	UpdatedAt   *time.Time      ``
}

func NewCard(title, description, status string, assignee, createdBy *members.Member) *Card {
	return &Card{
		Title:       title,
		Description: description,
		Status:      status,
		Assignee:    assignee,
		CreatedBy:   createdBy,
	}
}

func GetCard(id string) (*Card, error) {
	b, err := kanbanStore.Get(id)
	if err != nil {
		return nil, err
	}
	var card Card
	if err := json.Unmarshal(b, &card); err != nil {
		return nil, err
	}
	return &card, nil
}

func (c *Card) Save() error {
	b, err := c.toJson()
	if err != nil {
		return err
	}
	return kanbanStore.Upsert(c.Id, b)
}

func (c *Card) toJson() ([]byte, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	jsonMap := make(map[string]any)
	if err := json.Unmarshal(b, &jsonMap); err != nil {
		return nil, err
	}

	// convert created by and assignee to just member Ids
	if c.Assignee != nil {
		jsonMap["assignee"] = c.Assignee.Id
	}
	if c.CreatedBy != nil {
		jsonMap["created_by"] = c.CreatedBy.Id
	}

	return json.Marshal(jsonMap)
}
