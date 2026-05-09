package utils

import (
	"testing"
	"time"
)

func TestEnsureMinDuration(t *testing.T) {
	tests := []struct {
		name         string
		sleepBefore  time.Duration
		minDuration  time.Duration
		wantMinTotal time.Duration
	}{
		{
			name:         "already elapsed exceeds minimum",
			sleepBefore:  50 * time.Millisecond,
			minDuration:  10 * time.Millisecond,
			wantMinTotal: 50 * time.Millisecond,
		},
		{
			name:         "minimum not reached",
			sleepBefore:  0,
			minDuration:  50 * time.Millisecond,
			wantMinTotal: 50 * time.Millisecond,
		},
		{
			name:         "exactly minimum",
			sleepBefore:  30 * time.Millisecond,
			minDuration:  30 * time.Millisecond,
			wantMinTotal: 30 * time.Millisecond,
		},
		{
			name:         "zero minimum duration",
			sleepBefore:  10 * time.Millisecond,
			minDuration:  0,
			wantMinTotal: 10 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now().Add(-tt.sleepBefore)
			before := time.Now()
			EnsureMinDuration(start, tt.minDuration)
			elapsed := time.Since(before)
			totalElapsed := time.Since(start)

			if totalElapsed < tt.wantMinTotal {
				t.Errorf("total elapsed %v < expected minimum %v", totalElapsed, tt.wantMinTotal)
			}

			// If sleepBefore < minDuration, we expect some sleep to happen
			if tt.sleepBefore < tt.minDuration && tt.minDuration > 0 {
				expectedSleep := tt.minDuration - tt.sleepBefore
				// Allow some tolerance for scheduling variance
				if elapsed < expectedSleep-time.Millisecond {
					t.Errorf("expected at least %v sleep, got %v", expectedSleep, elapsed)
				}
			}

			_ = elapsed
		})
	}
}
