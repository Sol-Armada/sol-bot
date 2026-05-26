package tokens

import (
	"context"
	"errors"
	"time"
)

func Watch(ctx context.Context, out chan TokenRecord) error {
	if tokenStore == nil {
		return errors.New("token store not found")
	}
	lastRecordTS := time.Now().UTC()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(out)
			return nil
		case <-ticker.C:
			records, err := tokenStore.ListSince(lastRecordTS)
			if err != nil {
				return err
			}
			for _, r := range records {
				out <- r
				if r.CreatedAt.After(lastRecordTS) {
					lastRecordTS = r.CreatedAt
				}
			}
		}
	}
}
