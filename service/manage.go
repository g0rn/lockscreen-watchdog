// +build windows

package main

import (
	"fmt"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func startService(name string) error {
	manager, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer manager.Disconnect()

	service, err := manager.OpenService(name)
	if err != nil {
		return err
	}
	defer service.Close()

	err = service.Start()
	if err != nil {
		return err
	}

	return nil
}

func controlService(name string, cmd svc.Cmd, expectedState svc.State) error {
	manager, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer manager.Disconnect()

	service, err := manager.OpenService(name)
	if err != nil {
		return err
	}
	defer service.Close()

	status, err := service.Control(cmd)
	if err != nil {
		return err
	}

	timeout := time.Now().Add(10 * time.Second)
	for status.State != expectedState {
		if timeout.Before(timeout) {
			return fmt.Errorf("timeout waiting for service to change state to: %v", expectedState)
		}

		time.Sleep(300 * time.Millisecond)
		status, err = service.Query()
		if err != nil {
			return fmt.Errorf("can't query service status: %v", err)
		}
	}

	return nil
}
