package lib

import (
	"os/exec"
	"syscall"
)

func SetPlatformSpecificAttrs(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

func KillProcessGroup(cmd *exec.Cmd, signal syscall.Signal) error {
	return syscall.Kill(-cmd.Process.Pid, signal)
}
