package clock

import (
	"fmt"
	"log"
	"time"
)

// ResumeManager handles the logic for resuming clock state from Redis
type ResumeManager struct {
	clockRunner *ClockRunner
}

// NewResumeManager creates a new resume manager
func NewResumeManager(clockRunner *ClockRunner) *ResumeManager {
	return &ResumeManager{
		clockRunner: clockRunner,
	}
}

// ResumeFromRedis loads and resumes the system state from Redis
func (rm *ResumeManager) ResumeFromRedis() error {
	if rm.clockRunner.redisPersistence == nil {
		log.Printf("⚠️ Redis persistence not available, starting with default idle state")
		return rm.resetToIdle("redis not available")
	}

	state, err := rm.clockRunner.redisPersistence.LoadSystemState()
	if err != nil {
		log.Printf("❌ Failed to load system state from Redis: %v", err)
		log.Printf("🔄 Falling back to idle state due to Redis error")
		return rm.resetToIdle("redis load failed")
	}

	// Log the current state from Redis with more details
	rm.logResumeDebugInfo(state)

	// Check if the server has been offline for too long
	if rm.shouldResetDueToTimeout(state) {
		log.Printf("🔄 Server was offline for too long, resetting to idle")
		return rm.resetToIdle("server was offline for too long")
	}

	// Resume based on state
	if err := rm.resumeBasedOnState(state); err != nil {
		log.Printf("❌ Failed to resume based on state: %v", err)
		log.Printf("🔄 Falling back to idle state due to resume error")
		return rm.resetToIdle("resume failed")
	}

	return nil
}

// logResumeDebugInfo logs detailed information about the resume process
func (rm *ResumeManager) logResumeDebugInfo(state *SystemState) {
	log.Printf("=== RESUME DEBUG INFO ===")
	log.Printf("Loaded from Redis - Session: %d, State: %s, Timezone: %s",
		state.CurrentSession, state.State, state.Timezone)
	log.Printf("Running: %v, Paused: %v, TimeRemaining: %ds",
		state.IsRunning, state.IsPaused, state.TimeRemaining)
	log.Printf("End time: %s", state.EndTime.Format("2006-01-02 15:04:05"))

	serverTime := time.Now()
	log.Printf("Server time: %s", serverTime.Format("2006-01-02 15:04:05"))
	log.Printf("Time difference: %v", serverTime.Sub(state.EndTime))
}

// shouldResetDueToTimeout checks if the server was offline for too long
func (rm *ResumeManager) shouldResetDueToTimeout(state *SystemState) bool {
	serverTime := time.Now()
	if state.EndTime.Before(serverTime) {
		log.Printf("❌ Server was offline for too long (end time: %s, server time: %s). Resetting clock state.",
			state.EndTime.Format("2006-01-02 15:04:05"), serverTime.Format("2006-01-02 15:04:05"))
		return true
	}
	return false
}

// resetToIdle resets the clock to idle state
func (rm *ResumeManager) resetToIdle(reason string) error {
	log.Printf("🔄 Resetting to idle state: %s", reason)

	// Reset the clock state
	rm.clockRunner.stateManager.SetState(StateIdle)
	rm.clockRunner.sessionManager.ResetSessions()
	rm.clockRunner.timerManager.StopTimer()

	// Save the reset state to Redis
	if err := rm.clockRunner.persistenceManager.SaveSystemStateToRedis(); err != nil {
		log.Printf("Failed to save reset state to Redis: %v", err)
	}

	log.Printf("✅ Clock reset to idle state")
	return nil
}

// resumeBasedOnState resumes the clock based on the loaded state
func (rm *ResumeManager) resumeBasedOnState(state *SystemState) error {
	log.Printf("🔄 ResumeManager: resumeBasedOnState: Session: %d, State: '%s', Running: %v, Paused: %v, TimeRemaining: %dms",
		state.CurrentSession, state.State, state.IsRunning, state.IsPaused, state.TimeRemaining)

	// Synchronize state and timer before resuming
	rm.clockRunner.synchronizeStateTimer()

	// Validate the state first
	if err := rm.validateSystemState(state); err != nil {
		log.Printf("❌ Invalid state detected: %v", err)
		return rm.resetToIdle("invalid state detected")
	}

	// Handle idle state
	if state.State == string(StateIdle) {
		log.Printf("🔄 Restoring idle state")
		return rm.restoreIdleState()
	}

	// Handle active sessions (working, short_break, long_break)
	if state.State == string(StateWorking) || state.State == string(StateShortBreak) || state.State == string(StateLongBreak) {
		// Priority: paused > running > interrupted
		if state.IsPaused {
			return rm.resumePausedSession(state)
		} else if state.IsRunning {
			return rm.resumeRunningSession(state)
		} else {
			// Session is not running and not paused
			if state.TimeRemaining > 0 {
				// Has time remaining - treat as running session that should continue
				// This handles cases where the session state was saved but flags got reset
				return rm.resumeRunningSession(state)
			} else {
				// No time remaining - move to next session
				return rm.resumeCompletedSession(state)
			}
		}
	}

	// Unknown state - reset to idle
	log.Printf("🔄 Unknown state '%s' - resetting to idle", state.State)
	return rm.resetToIdle("unknown state")
}

// validateSystemState validates the loaded system state for consistency
func (rm *ResumeManager) validateSystemState(state *SystemState) error {
	if state == nil {
		return fmt.Errorf("system state is nil")
	}

	// Validate current session is within valid range
	totalSessions := rm.clockRunner.sessionManager.GetTotalSessions()
	if state.CurrentSession < 0 || state.CurrentSession >= totalSessions {
		return fmt.Errorf("current session %d is out of valid range [0, %d)", state.CurrentSession, totalSessions)
	}

	// Validate state string
	if state.State != string(StateIdle) && state.State != string(StateWorking) && state.State != string(StateShortBreak) && state.State != string(StateLongBreak) && state.State != string(StatePaused) {
		return fmt.Errorf("invalid state: %s", state.State)
	}

	// Validate state consistency
	if state.State == string(StateIdle) && (state.IsRunning || state.IsPaused || state.TimeRemaining > 0) {
		return fmt.Errorf("idle state should not have running/paused flags or remaining time")
	}

	if state.State == string(StatePaused) && !state.IsPaused {
		return fmt.Errorf("paused state should have IsPaused=true")
	}

	if state.IsRunning && state.IsPaused {
		return fmt.Errorf("cannot be both running and paused")
	}

	return nil
}

// restoreIdleState restores the clock to idle state
func (rm *ResumeManager) restoreIdleState() error {
	rm.clockRunner.stateManager.SetState(StateIdle)
	rm.clockRunner.sessionManager.ResetSessions()
	rm.clockRunner.timerManager.StopTimer()
	return nil
}

// resumeRunningSession resumes a running session
func (rm *ResumeManager) resumeRunningSession(state *SystemState) error {
	log.Printf("▶️ Resuming running session %d with %dms remaining", state.CurrentSession, state.TimeRemaining)

	// Set the session number and validate
	rm.clockRunner.sessionManager.SetCurrentSession(state.CurrentSession)

	// Set the state to the actual session state (working/short_break/long_break)
	clockState := ClockState(state.State)
	rm.clockRunner.stateManager.SetState(clockState)

	// Reset tick counter (no longer needed with new approach)

	// Calculate remaining time (from milliseconds)
	endTime := state.EndTime
	remainingTime := endTime.Sub(time.Now())

	// Handle case where session has completed while server was down
	if remainingTime <= 0 {
		log.Printf("⚡ Session completed while server was down, moving to next session")
		rm.handleCompletedSession(clockState)
		return nil
	}

	// Start timer with remaining time
	log.Printf("⏱️ Starting timer with %v remaining", remainingTime)
	err := rm.startTimerWithCallbacks(remainingTime, clockState)
	if err != nil {
		return err
	}

	// Start periodic Redis saves for resumed running session
	rm.clockRunner.runSaveStateToRedis()

	// Ensure the resumed state is saved to Redis with updated end time
	rm.clockRunner.saveStateToRedis()
	return nil
}

// resumePausedSession resumes a paused session
func (rm *ResumeManager) resumePausedSession(state *SystemState) error {
	log.Printf("⏸️ Resuming paused session %d with %dms remaining", state.CurrentSession, state.TimeRemaining)

	// Set the session number
	rm.clockRunner.sessionManager.SetCurrentSession(state.CurrentSession)

	// For paused sessions, set state to paused (not the session state)
	rm.clockRunner.stateManager.SetState(StatePaused)

	// Get the session state for timer setup
	sessionState := ClockState(state.State)

	// Calculate remaining time (from milliseconds)
	endTime := state.EndTime
	remainingTime := endTime.Sub(time.Now())

	// If there's time remaining, start the timer but pause it immediately
	if remainingTime > 0 {
		log.Printf("⏱️ Setting up paused timer with %v remaining", remainingTime)
		if err := rm.startTimerWithCallbacks(remainingTime, sessionState); err != nil {
			return err
		}
		rm.clockRunner.timerManager.PauseTimer()
		log.Printf("✅ Successfully restored paused session with %v remaining", remainingTime)
	} else {
		log.Printf("⚠️ No time remaining for paused session, will complete when resumed")
	}

	// Start periodic Redis saves for resumed paused session
	rm.clockRunner.runSaveStateToRedis()

	return nil
}

// resumeInterruptedSession handles sessions that were interrupted (not running, not paused)
func (rm *ResumeManager) resumeInterruptedSession(state *SystemState) error {
	log.Printf("⚡ Resuming interrupted session %d with %dms remaining", state.CurrentSession, state.TimeRemaining)

	// Set the session number
	rm.clockRunner.sessionManager.SetCurrentSession(state.CurrentSession)

	// Set the state to the actual session state
	clockState := ClockState(state.State)
	rm.clockRunner.stateManager.SetState(StatePaused) // Treat as paused for safety

	// Reset tick counter (no longer needed with new approach)

	// Calculate remaining time (from milliseconds)
	endTime := state.EndTime
	remainingTime := endTime.Sub(time.Now())

	// If there's time remaining, set up the timer but keep it paused
	if remainingTime > 0 {
		log.Printf("⏱️ Setting up interrupted session with %v remaining (paused)", remainingTime)
		if err := rm.startTimerWithCallbacks(remainingTime, clockState); err != nil {
			return err
		}
		rm.clockRunner.timerManager.PauseTimer()
		log.Printf("✅ Successfully restored interrupted session with %v remaining", remainingTime)
	} else {
		log.Printf("⚠️ Interrupted session had no time remaining, treating as paused with full duration")
		// Get the full duration for this session type
		sessionDuration := rm.clockRunner.sessionManager.GetCurrentSessionDuration()
		if sessionDuration > 0 {
			if err := rm.startTimerWithCallbacks(sessionDuration, clockState); err != nil {
				return err
			}
			rm.clockRunner.timerManager.PauseTimer()
		}
	}

	// Start periodic Redis saves for resumed interrupted session
	rm.clockRunner.runSaveStateToRedis()

	return nil
}

// resumeCompletedSession handles sessions that completed while server was down
func (rm *ResumeManager) resumeCompletedSession(state *SystemState) error {
	log.Printf("⚡ Resuming completed session %d", state.CurrentSession)

	// Set the session number
	rm.clockRunner.sessionManager.SetCurrentSession(state.CurrentSession)

	// Mark the session as completed and move to next
	clockState := ClockState(state.State)
	rm.handleCompletedSession(clockState)

	return nil
}

// handleCompletedSession handles when a session completed while server was down
func (rm *ResumeManager) handleCompletedSession(completedState ClockState) {
	// Record the completed session
	duration := rm.clockRunner.sessionManager.GetCurrentSessionDuration()
	rm.clockRunner.statsManager.RecordSession(completedState, duration)

	// Move to next session
	if !rm.clockRunner.sessionManager.NextSession() {
		// Completed all sessions
		rm.clockRunner.stateManager.SetState(StateIdle)
		if rm.clockRunner.onStateChange != nil {
			rm.clockRunner.onStateChange(StateIdle)
		}
		log.Printf("✅ All sessions completed while server was down")
		return
	}

	// Start next session
	rm.clockRunner.startNewSession()
}

// startTimerWithCallbacks starts the timer with proper callbacks
func (rm *ResumeManager) startTimerWithCallbacks(remainingTime time.Duration, clockState ClockState) error {
	log.Printf("▶️ ResumeManager: startTimerWithCallbacks: remainingTime=%v, clockState=%v", remainingTime, clockState)
	if remainingTime <= 0 {
		return fmt.Errorf("remaining time must be positive, got %v", remainingTime)
	}

	// Set up timer callbacks
	onTick := func(remaining time.Duration) {
		onTick(rm.clockRunner, remaining)
	}

	onComplete := func(completedState ClockState) {
		log.Printf("✅ Session completed: %s", completedState)

		// Record the completed session
		duration := rm.clockRunner.sessionManager.GetCurrentSessionDuration()
		rm.clockRunner.statsManager.RecordSession(completedState, duration)

		if rm.clockRunner.onComplete != nil {
			rm.clockRunner.onComplete(completedState)
		}

		// Move to next session
		if !rm.clockRunner.sessionManager.NextSession() {
			// Completed all sessions
			rm.clockRunner.stateManager.SetState(StateIdle)
			if rm.clockRunner.onStateChange != nil {
				rm.clockRunner.onStateChange(StateIdle)
			}
			log.Printf("🎉 All pomodoro sessions completed!")
			return
		}

		// Start next session
		rm.clockRunner.startNewSession()
	}

	// Start the timer with remaining time
	rm.clockRunner.timerManager.StartTimer(remainingTime, clockState, onTick, onComplete)
	log.Printf("⏱️ Started timer with %v remaining for %s session", remainingTime, clockState)

	return nil
}
