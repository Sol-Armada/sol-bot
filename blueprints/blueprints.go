package blueprints

import (
	"embed"
	"strings"
)

//go:embed blueprints
var blueprintsFS embed.FS

func List() ([]string, error) {
	entriesBytes, err := blueprintsFS.ReadFile("blueprints")
	if err != nil {
		return nil, err
	}

	entries := strings.Split(string(entriesBytes), "\n")

	blueprintNames := []string{}

	for _, entry := range entries {
		if entry != "" {
			blueprintNames = append(blueprintNames, entry)
		}
	}

	return blueprintNames, nil
}
