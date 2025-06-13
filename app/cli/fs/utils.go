package fs

import (
	"fmt"
	"os"
)

func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, fmt.Errorf("error checking if file exists: %v", err)
	}
}
