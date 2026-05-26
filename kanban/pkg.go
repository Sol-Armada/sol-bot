package kanban

import "errors"

type memoryKanbanStore struct {
	data map[string][]byte
}

func (s *memoryKanbanStore) Get(id string) ([]byte, error) {
	b, ok := s.data[id]
	if !ok {
		return nil, errors.New("kanban card not found")
	}
	return b, nil
}

func (s *memoryKanbanStore) Upsert(id string, payload []byte) error {
	s.data[id] = payload
	return nil
}

var kanbanStore = &memoryKanbanStore{data: map[string][]byte{}}
