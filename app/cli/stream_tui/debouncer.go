package streamtui

import (
	"sync"
	"time"
)

// UpdateDebouncer helps prevent visual glitches from rapid updates
type UpdateDebouncer struct {
	mu          sync.Mutex
	lastUpdate  time.Time
	minInterval time.Duration
	pending     bool
}

func NewUpdateDebouncer(minInterval time.Duration) *UpdateDebouncer {
	return &UpdateDebouncer{
		minInterval: minInterval,
	}
}

// ShouldUpdate returns true if enough time has passed since the last update
func (d *UpdateDebouncer) ShouldUpdate() bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	if now.Sub(d.lastUpdate) < d.minInterval {
		d.pending = true
		return false
	}

	d.lastUpdate = now
	d.pending = false
	return true
}
