// +build windows

package main

import (
	"fmt"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
)

var elog debug.Log

type service struct{}

func (s *service) Execute(args []string, request <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
loop:
	for change := range request {
		switch change.Cmd {
		case svc.Interrogate:
			changes <- change.CurrentStatus
		case svc.Stop, svc.Shutdown:
			break loop
		case svc.Pause:
			changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
		case svc.Continue:
			changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
		default:
			elog.Error(1, fmt.Sprintf("unexpected control request: #%d", change))
		}
	}

	changes <- svc.Status{State: svc.StopPending}
	return
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
