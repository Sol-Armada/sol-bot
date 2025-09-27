package bot

import (
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

	keys := make([]string, 0, len(messages.messages))
	for k := range messages.messages {
		keys = append(keys, k)
	}

	defer func() { currentMessageIndex = (currentMessageIndex + 1) % len(keys) }()
	return messages.messages[keys[currentMessageIndex]]
}

func removeStatusMessage(id string) {
	messages.mu.Lock()
	defer messages.mu.Unlock()
	delete(messages.messages, id)
}
