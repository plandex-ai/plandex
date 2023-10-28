package shared

import "time"

func StringTs() string {
	return time.Now().Format("2006-01-02T15:04:05.999Z")
}
