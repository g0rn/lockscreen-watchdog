// +build windows
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

func installService(name, descr string) error {
	manager, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer manager.Disconnect()

	service, err := manager.OpenService(name)
	if err == nil {
		service.Close()
		return fmt.Errorf("service %v already exists", name)
	}

	exePath, err := filepath.Abs(os.Args[0])
	if err != nil {
		return err
	}

	const SERVICE_AUTO_START = 2
	service, err = manager.CreateService(name, exePath,
		mgr.Config{
			Description: descr,
			StartType:   SERVICE_AUTO_START})
	if err != nil {
		return err
	}
	defer service.Close()

	err = eventlog.InstallAsEventCreate(name, eventlog.Error|eventlog.Info|eventlog.Warning)
	if err != nil {
		service.Delete()
		return fmt.Errorf("can't setup event log source: %v", err)
	}

	return nil
}

func removeService(name string) error {
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

	err = service.Delete()
	if err != nil {
		return err
	}

	err = eventlog.Remove(name)
	if err != nil {
		return fmt.Errorf("can't remove event log source: %v", err)
	}

	return nil
}
