package services

import "sync"

var (
	dispatchHookMu      sync.RWMutex
	dispatchPendingHook func()
	connectedNodeHookMu sync.RWMutex
	connectedNodeHook   func() []int64
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

// SetConnectedNodeProvider registers a provider for currently connected node IDs.
func SetConnectedNodeProvider(hook func() []int64) {
	connectedNodeHookMu.Lock()
	connectedNodeHook = hook
	connectedNodeHookMu.Unlock()
}

// ConnectedNodeIDs returns currently connected nodes if provider is set.
func ConnectedNodeIDs() []int64 {
	connectedNodeHookMu.RLock()
	hook := connectedNodeHook
	connectedNodeHookMu.RUnlock()
	if hook == nil {
		return nil
	}
	return hook()
}
