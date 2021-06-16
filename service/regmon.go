// +build windows

package main

import (
	"context"
	"fmt"
	"log"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

type changeNotification struct {
	oldValue string
	newValue string
}

func monitorRegValue(root registry.Key, keyPath string, keyName string, ctx context.Context) (<-chan changeNotification, error) {
	ch := make(chan changeNotification)

	advapi32, err := syscall.LoadDLL("Advapi32.dll")
	if err != nil {
		return nil, fmt.Errorf("can't load Advapi32.dll, err: %v", err)
	}

	regNotifyChangeKeyValue, err := advapi32.FindProc("RegNotifyChangeKeyValue")
	if err != nil {
		return nil, fmt.Errorf("can't find RegNotifyChangeKeyValue function: %v", err)
	}

	key, err := registry.OpenKey(root, keyPath, syscall.KEY_NOTIFY|registry.QUERY_VALUE)
	if err != nil {
		return nil, fmt.Errorf("can't open key: %v, err: %v", keyPath, err)
	}

	oldValue, _, err := key.GetStringValue(keyName)
	if err != nil {
		return nil, fmt.Errorf("can't get key value, err: %v", err)
	}
	key.Close()

	go func() {
		const REG_NOTIFY_CHANGE_LAST_SET = 0x00000004
		const REG_NOTIFY_CHANGE_NAME = 0x00000001
		for {
			key.Close()
			key, err = registry.OpenKey(root, keyPath, syscall.KEY_NOTIFY|registry.QUERY_VALUE)
			if err != nil {
				log.Fatalf("can't open key: %v, err: %v", keyPath, err)
			}

			select {
			case <-ctx.Done():
				close(ch)
				return
			default:
				regNotifyChangeKeyValue.Call(uintptr(key), 0, REG_NOTIFY_CHANGE_LAST_SET|REG_NOTIFY_CHANGE_NAME, 0, 0)
				if ctx.Err() != nil { // check if context was canceled during wait
					continue
				}
				newValue, _, err := key.GetStringValue(keyName)
				if err != nil {
					ch <- changeNotification{oldValue: oldValue, newValue: ""}
				}

				if oldValue != newValue {
					ch <- changeNotification{oldValue: oldValue, newValue: newValue}
				}
			}
		}
	}()

	return ch, nil
}
