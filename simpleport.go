package coinharness

import "sync"

// LazyPortManager simply generates subsequent
// integers starting from the BasePort
type LazyPortManager struct {
	// Harnesses will subsequently reserve
	// network ports starting from the BasePort value
	BasePort     int
	offset       int
	registerLock sync.RWMutex
}

// ObtainPort returns prev returned value + 1
// starting from the BasePort
func (man *LazyPortManager) ObtainPort() (result int) {
	man.registerLock.Lock()
	defer man.registerLock.Unlock()
	result = man.BasePort + man.offset
	man.offset++
	return
}
