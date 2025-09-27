package kanban

import (
	"github.com/sol-armada/sol-bot/stores"
)

var kanbanStore *stores.KanbanStore

func init() {
	storesClient := stores.Get()
	ks, ok := storesClient.GetKanbanStore()
	if !ok {
		panic("kanban store not found")
	}
	kanbanStore = ks
}
