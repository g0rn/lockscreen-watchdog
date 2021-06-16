// +build windows

package main

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
)

const (
	policyRegKeyPath  = `SOFTWARE\Policies\Microsoft\Windows\Personalization`
	policyRegKeyValue = "LockScreenImage"
)

var elog debug.Log

type service struct{}

func (s *service) Execute(args []string, request <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}
	ctx, cancel := context.WithCancel(context.Background())
	go runWatchdog(ctx)
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
loop:
	for change := range request {
		switch change.Cmd {
		case svc.Interrogate:
			changes <- change.CurrentStatus
		case svc.Stop, svc.Shutdown:
			break loop
		case svc.Pause:
			cancel()
			changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
		case svc.Continue:
			ctx, cancel = context.WithCancel(context.Background())
			go runWatchdog(ctx)
			changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
		default:
			elog.Error(1, fmt.Sprintf("unexpected control request: #%d", change))
		}
	}

	changes <- svc.Status{State: svc.StopPending}
	cancel()
	return
}

func runWatchdog(ctx context.Context) {
	elog.Info(1, "Starting Watchdog.")
	ch, err := monitorRegValue(registry.LOCAL_MACHINE, policyRegKeyPath, policyRegKeyValue, ctx)
	if err != nil {
		elog.Error(1, fmt.Sprintf("can't monitor registry key: %v", err))
	}

	for {
		select {
		case <-ctx.Done():
			elog.Info(1, "Stopping Watchdog.")
			return
		case change := <-ch:
			elog.Info(1, fmt.Sprintf("Change detected. %v -> %v", change.oldValue, change.newValue))
			time.AfterFunc(5*time.Second, func() {
				key, err := registry.OpenKey(registry.LOCAL_MACHINE, policyRegKeyPath, registry.SET_VALUE)
				if err != nil {
					elog.Error(1, fmt.Sprintf("Can't open registry key for editing: %v", err))
				}

				key.SetStringValue(policyRegKeyValue, change.oldValue)
				key.Close()
				elog.Info(1, fmt.Sprintf("Reverted: %v -> %v", change.newValue, change.oldValue))
			})
		}
	}
}

func runService(name string) {
	var err error
	elog, err = eventlog.Open(name)
	if err != nil {
		return
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("starting service %s", name))
	err = svc.Run(name, &service{})
	if err != nil {
		elog.Error(1, fmt.Sprintf("Can't run service: %v", err))
		return
	}

	elog.Info(1, fmt.Sprintf("service %v stopped", name))
}
