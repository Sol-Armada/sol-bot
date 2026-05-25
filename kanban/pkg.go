package kanban

import (
	"github.com/sol-armada/sol-bot/database/mongodb"
)

var kanbanStore *mongodb.KanbanStore

func init() {
	storesClient := mongodb.Get()
	ks, ok := storesClient.GetKanbanStore()
	if !ok {
		panic("kanban store not found")
	}
	kanbanStore = ks
}
