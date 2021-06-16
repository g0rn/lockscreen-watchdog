// +build windows

package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/sys/windows/svc"
)

const (
	serviceName = "LockscreenWatchdog"
)

func usage(errmsg string) {
	fmt.Fprintf(os.Stderr,
		"%s\n\n"+
			"usage: %s <command>\n"+
			"       where <command> is one of\n"+
			"       install, remove, start, stop, status, pause or continue.\n",
		errmsg, os.Args[0])
}

func main() {
	inService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("Failed to determine if we're running in Windows service: %v", err)
	}

	if inService {
		runService(serviceName)
		return
	}

	if len(os.Args) != 2 {
		usage("invalid command line arguments")
		os.Exit(1)
	}

	cmd := strings.ToLower(os.Args[1])
	switch cmd {
	case "install":
		err = installService(serviceName, "Lockscreen Watchdog Service")
	case "remove":
		err = removeService(serviceName)
	case "start":
		err = startService(serviceName)
	case "stop":
		err = controlService(serviceName, svc.Stop, svc.Stopped)
	case "pause":
		err = controlService(serviceName, svc.Pause, svc.Paused)
	case "continue":
		err = controlService(serviceName, svc.Continue, svc.Running)
	default:
		usage("invalid command")
		os.Exit(2)
	}

	if err != nil {
		log.Fatalf("Failed to execute command: %v", err)
	}
}
