package term

import (
	"time"

	"github.com/briandowns/spinner"
)

const minDuration = 700 * time.Millisecond

var s = spinner.New(spinner.CharSets[33], 100*time.Millisecond)
var startedAt time.Time

func StartSpinner(msg string) {
	startedAt = time.Now()
	s.Prefix = msg + " "
	s.Start()
}

func StopSpinner() {
	elapsed := time.Since(startedAt)
	if elapsed < minDuration {
		time.Sleep(minDuration - elapsed)
	}

	s.Stop()
	ClearCurrentLine()
}
