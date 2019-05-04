//+build !linux !amd64
//+build !darwin

package ntpdshm

import "errors"

// Get gets the shared-memory-segment of NTPD.
func Get(units uint) (*SHM, error) {
	return nil, errors.New("does not support shm")
}
