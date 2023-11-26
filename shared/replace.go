package shared

import (
	"strings"
)

type Replacement struct {
	Id         string `json:"id"`
	Old        string `json:"old"`
	New        string `json:"new"`
	Summary    string `json:"summary"`
	Failed     bool   `json:"failed"`
	RejectedAt string `json:"rejectedAt"`
}

func ApplyReplacements(content string, replacements []*Replacement, setFailed bool) (string, bool) {
	updated := content
	lastInsertedIdx := 0

	allSucceeded := true

	for _, replacement := range replacements {
		pre := updated[:lastInsertedIdx]
		sub := updated[lastInsertedIdx:]
		originalIdx := strings.Index(sub, replacement.Old)

		if originalIdx == -1 {
			allSucceeded = false
			if setFailed {
				replacement.Failed = true
			}

			// jsonBytes, _ := json.Marshal(replacement)
			// log.Println(string(jsonBytes))

			// log.Println("Replacement: " + replacement.Old + " -> " + replacement.New)

		} else {
			replaced := strings.Replace(sub, replacement.Old, replacement.New, 1)

			// log.Println("Replacement: " + replacement.Old + " -> " + replacement.New)
			// log.Println("Pre: " + pre)
			// log.Println("Sub: " + sub)
			// log.Println("Idx: " + fmt.Sprintf("%d", idx))
			// log.Println("Updated: " + updated)

			updated = pre + replaced
			lastInsertedIdx = lastInsertedIdx + originalIdx + len(replacement.New)
		}
	}

	return updated, allSucceeded
}

func (rep *Replacement) IsPending() bool {
	return !rep.Failed && rep.RejectedAt == ""
}

func (rep *Replacement) SetRejected(ts string) {
	rep.RejectedAt = ts
}
