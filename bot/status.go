package bot

import (
	"maps"
	"slices"
	"sync"
)

type statusMessages struct {
	messages map[string]string

	mu sync.Mutex
}

var messages = statusMessages{messages: map[string]string{}}
var currentMessageIndex = 0

func upsertStatusMessage(id, message string) {
	messages.mu.Lock()
	defer messages.mu.Unlock()
	messages.messages[id] = message
}

func clearStatusMessages() {
	messages.mu.Lock()
	defer messages.mu.Unlock()
	messages.messages = map[string]string{}
	currentMessageIndex = 0
}

func NextStatusMessage() string {
	if len(messages.messages) == 0 {
		return ""
	}

	messages.mu.Lock()
	defer messages.mu.Unlock()

	keys := slices.Collect(maps.Keys(messages.messages))

	defer func() { currentMessageIndex = (currentMessageIndex + 1) % len(keys) }()

	if len(keys) == 0 || currentMessageIndex < 0 {
		return ""
	}

	return messages.messages[keys[currentMessageIndex]]
}

func removeStatusMessage(id string) {
	messages.mu.Lock()
	defer messages.mu.Unlock()
	delete(messages.messages, id)
}
