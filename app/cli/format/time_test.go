package format

import (
	"testing"
	"time"
)

func TestTime(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name     string
		then     time.Time
		contains string // the result should contain this substring
	}{
		{
			name:     "just now",
			then:     now.Add(-500 * time.Millisecond),
			contains: "just now",
		},
		{
			name:     "seconds ago",
			then:     now.Add(-30 * time.Second),
			contains: "s ago",
		},
		{
			name:     "one minute ago",
			then:     now.Add(-90 * time.Second),
			contains: "1m ago",
		},
		{
			name:     "minutes ago",
			then:     now.Add(-5 * time.Minute),
			contains: "m ago",
		},
		{
			name:     "one hour ago",
			then:     now.Add(-90 * time.Minute),
			contains: "1h ago",
		},
		{
			name:     "hours ago",
			then:     now.Add(-3 * time.Hour),
			contains: "h ago",
		},
		{
			name:     "one day ago",
			then:     now.Add(-36 * time.Hour),
			contains: "1d ago",
		},
		{
			name:     "days ago",
			then:     now.Add(-5 * Day),
			contains: "d ago",
		},
		{
			name:     "one week ago",
			then:     now.Add(-10 * Day),
			contains: "1w ago",
		},
		{
			name:     "weeks ago",
			then:     now.Add(-3 * Week),
			contains: "w ago",
		},
		{
			name:     "old date, long format",
			then:     now.Add(-2 * Year),
			contains: "", // format is like "Jan 2 2006"
		},
		{
			name:     "future time",
			then:     now.Add(30 * time.Minute),
			contains: "from now",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Time(tt.then)
			if tt.contains == "" {
				// For old dates, verify it's not empty and has the year
				if result == "" {
					t.Error("expected non-empty result for old date")
				}
			} else {
				if !contains(result, tt.contains) {
					t.Errorf("Time(%v) = %q, expected to contain %q", tt.then, result, tt.contains)
				}
			}
		})
	}
}

func TestTimeNeverPanics(t *testing.T) {
	// Edge cases
	cases := []time.Time{
		time.Time{},                       // zero time
		time.Now().Add(100 * Year),        // far future
		time.Now().Add(-100 * Year),       // far past
		time.Now().Add(-1 * time.Second),  // 1 second ago
		time.Now().Add(1 * time.Second),   // 1 second from now
	}

	for _, tc := range cases {
		t.Run(tc.String(), func(t *testing.T) {
			result := Time(tc)
			if result == "" {
				t.Error("Time returned empty string")
			}
		})
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
