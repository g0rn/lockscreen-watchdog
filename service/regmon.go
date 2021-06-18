// +build windows

package main

import (
	"context"
	"log"
	"syscall"
	"time"

	"golang.org/x/sys/windows/registry"
)

var (
	regNotifyChangeKeyValue *syscall.Proc
)

func init() {
	advapi32, err := syscall.LoadDLL("Advapi32.dll")
	if err != nil {
		log.Fatalf("Can't load Advapi32.dll %v", err)
	}

	regNotifyChangeKeyValue, err = advapi32.FindProc("RegNotifyChangeKeyValue")
	if err != nil {
		log.Fatalf("Can't find RegNotifyChangeKeyValue function in DLL: %v", err)
	}
}

type changeNotification struct {
	oldValue string
	newValue string
}

func readRegKeyString(root registry.Key, keyPath string, keyName string) (string, error) {
	key, err := registry.OpenKey(root, keyPath, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer key.Close()

	val, _, err := key.GetStringValue(keyName)
	return val, err
}

func waitRegKeyChanged(root registry.Key, keyPath string, keyName string) error {
	key, err := registry.OpenKey(root, keyPath, registry.QUERY_VALUE|syscall.KEY_NOTIFY)
	if err != nil {
		return err
	}
	defer key.Close()

	const REG_NOTIFY_CHANGE_LAST_SET = 0x00000004
	const REG_NOTIFY_CHANGE_NAME = 0x00000001
	regNotifyChangeKeyValue.Call(uintptr(key), 0, REG_NOTIFY_CHANGE_LAST_SET|REG_NOTIFY_CHANGE_NAME, 0, 0)
	return nil
}

func monitorRegValue(root registry.Key, keyPath string, keyName string, ctx context.Context) (<-chan changeNotification, error) {
	ch := make(chan changeNotification)

	oldValue, err := readRegKeyString(root, keyPath, keyName)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				close(ch)
				return
			default:
				err = waitRegKeyChanged(root, keyPath, keyName)
				if err != nil {
					log.Fatalf("waitRegKeyChanged failed: %v", err)
				}

				if ctx.Err() != nil { // check if context was canceled during wait
					continue
				}
				// Wait before reading new value
				time.Sleep(5 * time.Second)
				newValue, err := readRegKeyString(root, keyPath, keyName)
				if err != nil {
					ch <- changeNotification{oldValue: oldValue, newValue: ""}
					continue
				}

				if oldValue != newValue {
					ch <- changeNotification{oldValue: oldValue, newValue: newValue}
				}
			}
		}
	}()

	return ch, nil
}
