//go:build (linux && amd64) || darwin
// +build linux,amd64 darwin

package ntpdshm

import (
	"reflect"
	"syscall"
	"unsafe"
)

// Get gets the shared-memory-segment of NTPD.
func Get(units uint) (*SHM, error) {
	shmID, _, err := syscall.Syscall(syscall.SYS_SHMGET, uintptr(ntpdSHMKey+units), 0, 0600)
	if err != 0 {
		return nil, err
	}
	shm, _, err := syscall.Syscall(syscall.SYS_SHMAT, shmID, 0, 0)
	if err != 0 {
		return nil, err
	}

	var data []shmTime
	header := (*reflect.SliceHeader)(unsafe.Pointer(&data))
	header.Data = shm
	header.Len = 1
	header.Cap = 1
	return &SHM{data: data}, nil
}
