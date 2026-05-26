package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/sol-armada/sol-bot/database/sqlc/dbgen"
)

func GetAttendanceTags() ([]string, error) {
	q, err := queries()
	if err != nil {
		return nil, err
	}
	return q.ListAttendanceTags(context.Background())
}

func NewAttendanceTag(tag string) error {
	q, err := queries()
	if err != nil {
		return err
	}

	tag = normalizeConfigValue(tag)
	if tag == "" {
		return fmt.Errorf("attendance tag cannot be empty")
	}

	return q.InsertAttendanceTag(context.Background(), tag)
}

func GetAttendanceNames() ([]string, error) {
	q, err := queries()
	if err != nil {
		return nil, err
	}
	return q.ListAttendanceNames(context.Background())
}

func ValidAttendanceName(name string) (bool, error) {
	normalized := normalizeConfigValue(name)
	if normalized == "" {
		return false, nil
	}

	names, err := GetAttendanceNames()
	if err != nil {
		return false, err
	}

	for _, existing := range names {
		if strings.EqualFold(normalized, existing) {
			return true, nil
		}
	}

	return false, nil
}

func NewAttendanceName(name string) error {
	q, err := queries()
	if err != nil {
		return err
	}

	name = normalizeConfigValue(name)
	if name == "" {
		return fmt.Errorf("attendance name cannot be empty")
	}

	return q.InsertAttendanceName(context.Background(), name)
}

func RemoveAttendanceName(name string) error {
	q, err := queries()
	if err != nil {
		return err
	}

	normalized := normalizeConfigValue(name)
	if normalized == "" {
		return nil
	}

	names, err := q.ListAttendanceNames(context.Background())
	if err != nil {
		return err
	}
	for _, existing := range names {
		if strings.EqualFold(existing, normalized) {
			return q.DeleteAttendanceName(context.Background(), existing)
		}
	}

	return nil
}

func replaceAttendanceTags(tags []string) error {
	return withTx(func(qtx *dbgen.Queries) error {
		ctx := context.Background()
		if err := qtx.DeleteAllAttendanceTags(ctx); err != nil {
			return err
		}

		seen := map[string]struct{}{}
		for _, tag := range tags {
			normalized := normalizeConfigValue(tag)
			if normalized == "" {
				continue
			}
			if _, exists := seen[normalized]; exists {
				continue
			}
			seen[normalized] = struct{}{}
			if err := qtx.InsertAttendanceTag(ctx, normalized); err != nil {
				return err
			}
		}
		return nil
	})
}

func replaceAttendanceNames(names []string) error {
	return withTx(func(qtx *dbgen.Queries) error {
		ctx := context.Background()
		if err := qtx.DeleteAllAttendanceNames(ctx); err != nil {
			return err
		}

		seen := map[string]struct{}{}
		for _, name := range names {
			normalized := normalizeConfigValue(name)
			if normalized == "" {
				continue
			}
			if _, exists := seen[strings.ToLower(normalized)]; exists {
				continue
			}
			seen[strings.ToLower(normalized)] = struct{}{}
			if err := qtx.InsertAttendanceName(ctx, normalized); err != nil {
				return err
			}
		}
		return nil
	})
}

func normalizeConfigValue(value string) string {
	return strings.TrimSpace(value)
}
