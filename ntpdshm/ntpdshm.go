package ntpdshm

import (
	"sync"
	"sync/atomic"
	"time"
)

// Leap is a leap second indicator.
type Leap int

const (
	// LeapNoWarning indicates no leap second.
	LeapNoWarning Leap = 0

	// LeapAddSecond indicates a positive leap second.
	LeapAddSecond Leap = 1

	// LeapDelSecond indicates a negative leap second.
	LeapDelSecond Leap = 2

	// LeapNotInSync indicates a leap second.
	LeapNotInSync Leap = 3
)

func (l Leap) String() string {
	switch l {
	case LeapNoWarning:
		return "no leap warning"
	case LeapAddSecond:
		return "add leap second"
	case LeapDelSecond:
		return "del leap second"
	case LeapNotInSync:
		return "not in sync"
	}
	return ""
}

// SHM the shared-memory-segment of NTPD.
type SHM struct {
	mu   sync.Mutex
	data []shmTime
}

type shmTime struct {
	mode                 int32
	count                int32
	clockTimeStampSec    uint  // external clock
	clockTimeStampUSec   int32 // external clock
	receiveTimeStampSec  uint  // internal clock, when external value was received
	receiveTimeStampUSec int32 // internal clock, when external value was received
	leap                 int32
	precision            int32
	nsamples             int32
	valid                int32
	clockTimeStampNSec   uint32 // Unsigned ns timestamps
	receiveTimeStampNSec uint32 // Unsigned ns timestamps
	dummy                [8]uint32
}

const ntpdSHMKey = 0x4e545030

// Lock locks the SHM.
func (shm *SHM) Lock() {
	shm.mu.Lock()
	shm.SetValid(false)
}

// Unlock unlocks the SHM.
func (shm *SHM) Unlock() {
	shm.SetValid(true)
	shm.mu.Unlock()
}

// Mode returns the mode.
// 0 - if valid is set: use values, clear valid.
// 1 - if valid is set: if count before and after read of data is equal: use values, clear valid.
func (shm *SHM) Mode() int32 {
	return shm.data[0].mode
}

// SetMode set the mode.
func (shm *SHM) SetMode(mode int32) {
	shm.data[0].mode = mode
}

// Count returns the count.
func (shm *SHM) Count() int32 {
	return atomic.LoadInt32(&shm.data[0].count)
}

// SetCount sets the count.
func (shm *SHM) SetCount(count int32) {
	atomic.StoreInt32(&shm.data[0].count, count)
}

// IncrCount increments the count.
func (shm *SHM) IncrCount() {
	atomic.AddInt32(&shm.data[0].count, 1)
}

// Leap returns the leap indicator.
func (shm *SHM) Leap() Leap {
	return Leap(shm.data[0].leap)
}

// SetLeap sets the leap indicator.
func (shm *SHM) SetLeap(leap Leap) {
	shm.data[0].leap = int32(leap)
}

// Precision returns the precision.
func (shm *SHM) Precision() int32 {
	return shm.data[0].precision
}

// SetPrecision sets the precision.
func (shm *SHM) SetPrecision(precision int32) {
	shm.data[0].precision = precision
}

// NSamples returns the number of samples.
func (shm *SHM) NSamples() int32 {
	return shm.data[0].nsamples
}

// SetNSamples sets the number of samples.
func (shm *SHM) SetNSamples(nsamples int32) {
	shm.data[0].nsamples = nsamples
}

// Valid returns whether the SHM is valid.
func (shm *SHM) Valid() bool {
	return atomic.LoadInt32(&shm.data[0].valid) != 0
}

// SetValid sets whether the SHM is valid.
func (shm *SHM) SetValid(valid bool) {
	if valid {
		atomic.StoreInt32(&shm.data[0].valid, 1)
	} else {
		atomic.StoreInt32(&shm.data[0].valid, 0)
	}
}

// ClockTimeStamp returns the clock timestamp.
func (shm *SHM) ClockTimeStamp() time.Time {
	return time.Unix(int64(shm.data[0].clockTimeStampSec), int64(shm.data[0].clockTimeStampNSec))
}

// SetClockTimeStamp sets the clock timestamp.
func (shm *SHM) SetClockTimeStamp(t time.Time) {
	sec := t.Unix()
	nsec := t.Nanosecond()
	shm.data[0].clockTimeStampSec = uint(sec)
	shm.data[0].clockTimeStampUSec = int32(nsec / 1e3)
	shm.data[0].clockTimeStampNSec = uint32(nsec)
}

// ReceiveTimeStamp returns the receive timestamp.
func (shm *SHM) ReceiveTimeStamp() time.Time {
	return time.Unix(int64(shm.data[0].receiveTimeStampSec), int64(shm.data[0].receiveTimeStampNSec))
}

// SetReceiveTimeStamp sets the receive timestamp.
func (shm *SHM) SetReceiveTimeStamp(t time.Time) {
	sec := t.Unix()
	nsec := t.Nanosecond()
	shm.data[0].receiveTimeStampSec = uint(sec)
	shm.data[0].receiveTimeStampUSec = int32(nsec / 1e3)
	shm.data[0].receiveTimeStampNSec = uint32(nsec)
}
