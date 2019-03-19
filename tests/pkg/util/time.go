package util

import (
	"context"
	"time"
)

// Sleep defines special `sleep` with context
func Sleep(ctx context.Context, sleepTime time.Duration) {
	ticker := time.NewTicker(sleepTime)
	defer ticker.Stop()

	select {
	case <-ctx.Done():
		return
	case <-ticker.C:
		return
	}
}
