// Adapted from https://raw.githubusercontent.com/dustin/go-humanize/master/times.go

package format

import (
	"fmt"
	"sort"
	"time"
)

// Seconds-based time units
const (
	Day      = 24 * time.Hour
	Week     = 7 * Day
	Month    = 30 * Day
	Year     = 12 * Month
	LongTime = 37 * Year
)

// Time formats a time into a relative string.
//
// Time(someT) -> "3 weeks ago"
func Time(then time.Time) string {
	return relTime(then.UTC(), time.Now().UTC(), "ago", "from now")
}

// A relTimeMagnitude struct contains a relative time point at which
// the relative format of time will switch to a new format string.  A
// slice of these in ascending order by their "D" field is passed to
// CustomRelTime to format durations.
//
// The Format field is a string that may contain a "%s" which will be
// replaced with the appropriate signed label (e.g. "ago" or "from
// now") and a "%d" that will be replaced by the quantity.
//
// The DivBy field is the amount of time the time difference must be
// divided by in order to display correctly.
//
// e.g. if D is 2*time.Minute and you want to display "%d minutes %s"
// DivBy should be time.Minute so whatever the duration is will be
// expressed in minutes.
type relTimeMagnitude struct {
	D      time.Duration
	Format string
	DivBy  time.Duration
}

var defaultMagnitudes = []relTimeMagnitude{
	{time.Second, "just now", time.Second},
	{2 * time.Second, "1s %s", 1},
	{time.Minute, "%ds %s", time.Second},
	{2 * time.Minute, "1m %s", 1},
	{time.Hour, "%dm %s", time.Minute},
	{2 * time.Hour, "1h %s", 1},
	{Day, "%dh %s", time.Hour},
	{2 * Day, "1d %s", 1},
	{Week, "%dd %s", Day},
	{2 * Week, "1w %s", 1},
	{Month, "%dw %s", Week},
}

// RelTime(timeInPast, timeInFuture, "earlier", "later") -> "3 weeks earlier"
func relTime(a, b time.Time, albl, blbl string) string {
	return customRelTime(a, b, albl, blbl, defaultMagnitudes)
}

func customRelTime(a, b time.Time, albl, blbl string, magnitudes []relTimeMagnitude) string {
	lbl := albl
	diff := b.Sub(a)

	if a.After(b) {
		lbl = blbl
		diff = a.Sub(b)
	}

	// Find the largest magnitude
	largestMagnitude := magnitudes[len(magnitudes)-1].D

	// If the difference is greater than the largest magnitude, format the date in local time
	if diff >= largestMagnitude {
		return a.Local().Format("Jan 2 2006")
	}

	n := sort.Search(len(magnitudes), func(i int) bool {
		return magnitudes[i].D > diff
	})

	// If no magnitude is large enough, use the largest magnitude available
	if n >= len(magnitudes) {
		n = len(magnitudes) - 1
	}
	mag := magnitudes[n]

	if mag.DivBy == 1 {
		return fmt.Sprintf(mag.Format, lbl)
	}

	return fmt.Sprintf(mag.Format, diff/mag.DivBy, lbl)
}
