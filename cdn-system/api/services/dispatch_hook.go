package services

import "sync"

var (
	dispatchHookMu      sync.RWMutex
	dispatchPendingHook func()
)

// SetDispatchPendingHook registers a dispatcher callback for new tasks/jobs.
func SetDispatchPendingHook(hook func()) {
	dispatchHookMu.Lock()
	dispatchPendingHook = hook
	dispatchHookMu.Unlock()
}

// TriggerDispatchPending signals the dispatcher to push pending work.
func TriggerDispatchPending() {
	dispatchHookMu.RLock()
	hook := dispatchPendingHook
	dispatchHookMu.RUnlock()
	if hook != nil {
		hook()
	}
}
