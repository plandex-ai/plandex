package term

import (
	"fmt"
	"os"
	"runtime"
)

func GetOsDetails() string {
	return fmt.Sprintf(
		"OS: %s\nArchitecture: %s\nCPUs: %d\nShell: %s",
		runtime.GOOS,
		runtime.GOARCH,
		runtime.NumCPU(),
		os.Getenv("SHELL"),
	)
}
