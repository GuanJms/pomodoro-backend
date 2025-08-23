package clock

import (
	"fmt"
	"sync"
)

// StateManager handles clock state transitions and validation
type StateManager struct {
	mu    sync.RWMutex
	state ClockState
}

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
	return &StateManager{
		state: StateIdle,
	}
}

// GetState returns the current state
func (sm *StateManager) GetState() ClockState {
	// No lock needed - in polling scenarios, slightly stale reads are acceptable
	// and write operations should have priority
	return sm.state
}

// SetState sets the current state
func (sm *StateManager) SetState(state ClockState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.state = state
}

// IsIdle returns true if the clock is idle
func (sm *StateManager) IsIdle() bool {
	return sm.GetState() == StateIdle
}

// IsRunning returns true if the clock is actively running
func (sm *StateManager) IsRunning() bool {
	state := sm.GetState()
	return state == StateWorking || state == StateShortBreak || state == StateLongBreak
}

// IsPaused returns true if the clock is paused
func (sm *StateManager) IsPaused() bool {
	return sm.GetState() == StatePaused
}

// CanStart returns true if the clock can be started
func (sm *StateManager) CanStart() bool {
	state := sm.GetState()
	return state == StateIdle || state == StatePaused
}

// CanPause returns true if the clock can be paused
func (sm *StateManager) CanPause() bool {
	return sm.IsRunning()
}

// CanStop returns true if the clock can be stopped
func (sm *StateManager) CanStop() bool {
	return !sm.IsIdle()
}

// CanSkip returns true if the clock can be skipped
func (sm *StateManager) CanSkip() bool {
	return sm.IsRunning() || sm.IsPaused()
}

// ValidateTransition validates if a state transition is allowed
func (sm *StateManager) ValidateTransition(newState ClockState) error {
	currentState := sm.GetState()

	switch newState {
	case StateIdle:
		// Can transition to idle from any state
		return nil
	case StateWorking, StateShortBreak, StateLongBreak:
		// Can only transition to running states from idle or paused
		if currentState != StateIdle && currentState != StatePaused {
			return fmt.Errorf("cannot transition from %s to %s", currentState, newState)
		}
		return nil
	case StatePaused:
		// Can only pause from running states
		if !sm.IsRunning() {
			return fmt.Errorf("cannot pause from state %s", currentState)
		}
		return nil
	default:
		return fmt.Errorf("invalid state: %s", newState)
	}
}
