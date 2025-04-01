//go:build !linux
// +build !linux

package lib

import "os/exec"

func MaybeIsolateCgroup(cmd *exec.Cmd) (deleteFn func()) {
	return func() {}
}
