//go:build windows

package lib

import (
	"os/exec"
	"strconv"
	"syscall"

	"golang.org/x/sys/windows"
)

func SetPlatformSpecificAttrs(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP,
	}
}

func KillProcessGroup(cmd *exec.Cmd, signal syscall.Signal) error {
	// Windows uses different signals
	if signal == syscall.SIGINT {
		// Ctrl+C event
		return windows.GenerateConsoleCtrlEvent(windows.CTRL_C_EVENT, uint32(cmd.Process.Pid))
	}
	// For SIGKILL equivalent, terminate the process tree
	return exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
}
