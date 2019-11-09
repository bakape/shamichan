package util

import "sync"

var (
	hooks   = make(map[string][]func() error)
	hooksMu sync.RWMutex
)

// Hook a function to execute on an event
func Hook(event string, fn func() error) {
	hooksMu.Lock()
	defer hooksMu.Unlock()

	hooks[event] = append(hooks[event], fn)
}

// Trigger all hooks for specified event
func Trigger(event string) (err error) {
	hooksMu.RLock()
	defer hooksMu.RUnlock()

	for _, f := range hooks[event] {
		err = f()
		if err != nil {
			return
		}
	}
	return
}
