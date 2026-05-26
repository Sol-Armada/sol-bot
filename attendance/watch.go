package attendance

import (
	"context"
	"time"
)

func Watch(ctx context.Context, out chan Attendance) error {
	seen := map[string]time.Time{}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(out)
			return nil
		case <-ticker.C:
			attendances, err := List(nil, 0, 0)
			if err != nil {
				return err
			}
			for _, a := range attendances {
				if a == nil {
					continue
				}
				last, ok := seen[a.Id]
				if !ok || a.DateUpdated.After(last) {
					seen[a.Id] = a.DateUpdated
					out <- *a
				}
			}
		}
	}
}
