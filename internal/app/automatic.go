package app

import (
	"context"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/amxv/noria/internal/config"
	"github.com/amxv/noria/internal/host"
	"github.com/amxv/noria/internal/service"
)

// The daemon owns automatic updates even when the menu bar is not running.
// The child is the updater command, never a model or delegated coding agent.
func startAutomaticUpdates(ctx context.Context, home string, workers *sync.WaitGroup) {
	workers.Add(1)
	go func() {
		defer workers.Done()
		timer := time.NewTimer(2 * time.Minute)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
			}
			c, e := config.Load(home)
			if e == nil && c.AutoUpdate {
				exe := service.ReadIntent(home).Executable
				if exe != "" {
					command := exec.Command(exe, "update", "--automatic", "--json", "--home", home)
					command.Stdin = nil
					command.Stdout = io.Discard
					command.Stderr = io.Discard
					command.Env = os.Environ()
					// Do not cancel an updater because it requested the daemon's graceful
					// restart after swapping binaries. Each updater has its own deadline.
					host.Detach(command)
					if command.Start() == nil {
						exited := make(chan struct{})
						go func() { _ = command.Wait(); close(exited) }()
						select {
						case <-ctx.Done():
							return
						case <-exited:
						}
					}
				}
			}
			timer.Reset(15 * time.Minute)
		}
	}()
}
