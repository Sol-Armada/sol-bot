package kanban

import (
	"encoding/json"
	"time"

	"github.com/sol-armada/sol-bot/members"
	"go.mongodb.org/mongo-driver/bson"
)

type Card struct {
	Id          string          `bson:"_id"`
	Title       string          `bson:"title"`
	Description string          `bson:"description"`
	Status      string          `bson:"status"`
	Assignee    *members.Member `bson:"assignee"`
	CreatedBy   *members.Member `bson:"created_by"`
	CreatedAt   *time.Time      `bson:"created_at"`
	UpdatedAt   *time.Time      `bson:"updated_at"`
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

	return bson.Marshal(jsonMap)
}
