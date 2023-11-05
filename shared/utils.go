package shared

import "time"

const TsFormat = "2006-01-02T15:04:05.999Z"

func StringTs() string {
	return time.Now().UTC().Format(TsFormat)
}
