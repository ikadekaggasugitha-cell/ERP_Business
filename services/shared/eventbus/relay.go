package eventbus

import (
	"context"
	"log"
	"time"
)

// RunOutboxRelay polls drainStep on a fixed tick (and drains backlogs immediately)
// until ctx is cancelled.
func RunOutboxRelay(ctx context.Context, service string, interval time.Duration, drainStep func(context.Context) (int, error)) {
	if interval <= 0 {
		interval = 300 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		for {
			n, err := drainStep(ctx)
			if err != nil {
				if ctx.Err() == nil {
					log.Printf("[%s-relay] drain error: %v", service, err)
				}
				break
			}
			if n == 0 || ctx.Err() != nil {
				break
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

