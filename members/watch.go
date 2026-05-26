package members

import (
	"context"
	"time"
)

func Watch(ctx context.Context, out chan Member) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	seen := map[string]struct{}{}

	ids, err := GetStoredMemberIDs()
	if err != nil {
		return err
	}
	for _, id := range ids {
		seen[id] = struct{}{}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		ids, err := GetStoredMemberIDs()
		if err != nil {
			return err
		}

		for _, id := range ids {
			if _, ok := seen[id]; ok {
				continue
			}

			member, err := Get(id)
			if err != nil {
				return err
			}
			out <- *member
			seen[id] = struct{}{}
		}
	}
}
