//go:build linux
// +build linux

package lib

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"time"

	systemdDbus "github.com/coreos/go-systemd/v22/dbus"
	"github.com/godbus/dbus/v5"
	"github.com/google/uuid"
)

const cgroupCallTimeout = 1 * time.Second

func MaybeIsolateCgroup(cmd *exec.Cmd) (deleteFn func()) {
	noop := func() {}
	pid := cmd.Process.Pid

	// 1. Connect to the user manager (no prompt on typical distros).
	ctx, _ := context.WithTimeout(context.Background(), cgroupCallTimeout)

	conn, err := systemdDbus.NewUserConnectionContext(ctx)
	if err != nil {
		log.Printf("⚠️  Could not connect to user systemd manager. No cgroup isolation for PID %d. Error: %v", pid, err)
		return noop
	}
	// We'll keep 'conn' open while scope is active. The scope isn't strictly tied
	// to the connection's lifetime, but it's nice to keep it in case we want to stop the unit.

	scopeName := fmt.Sprintf("plandex-%s.scope", uuid.New().String())

	props := []systemdDbus.Property{
		systemdDbus.PropDescription("Plandex user-scope isolation"),
		// Under system manager, user.slice means “treat it as a user process.”
		// If this is truly in the user manager, it may ignore the slice or map it differently.
		systemdDbus.PropSlice("user.slice"),

		// KillMode=control-group: stopping the scope kills all processes in cgroup.
		systemdDbus.Property{Name: "KillMode", Value: dbus.MakeVariant("control-group")},

		// Attach the existing process by PID
		systemdDbus.PropPids(uint32(pid)),

		// Optional: auto-remove the scope once no processes remain.
		systemdDbus.Property{Name: "CollectMode", Value: dbus.MakeVariant("inactive-or-failed")},
	}

	_, err = conn.StartTransientUnitContext(ctx, scopeName, "replace", props, nil)
	if err != nil {
		// Fallback, no isolation
		log.Printf("⚠️  Failed to start transient scope for PID %d: %v", pid, err)
		return noop
	}

	return func() {
		// Close the connection to the user manager.
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), cgroupCallTimeout)
		defer cancel()

		// Attempt to stop the scope (killing all processes if any remain).
		_, stopErr := conn.StopUnitContext(ctx, scopeName, "replace", nil)
		if stopErr != nil {
			log.Printf("⚠️  Failed to stop scope %s: %v", scopeName, stopErr)
		}
	}
}
