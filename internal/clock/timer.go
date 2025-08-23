package clock

import (
	"context"
	"log"
	"sync"
	"time"
)

// TimerManager handles the actual timing logic and ticker functionality
type TimerManager struct {
	mu sync.RWMutex

	// Timer state
	timer  *time.Timer
	ticker *time.Ticker
	ctx    context.Context
	cancel context.CancelFunc

	// Timing info
	startTime       time.Time
	pauseTime       time.Time
	timeRemaining   time.Duration
	sessionDuration time.Duration
	currentState    ClockState

	// Callbacks
	onTick     func(time.Duration)
	onComplete func(ClockState)
}

// NewTimerManager creates a new timer manager
func NewTimerManager() *TimerManager {
	return &TimerManager{}
}

// StartTimer starts the timer for a session
func (tm *TimerManager) StartTimer(duration time.Duration, state ClockState, onTick func(time.Duration), onComplete func(ClockState)) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Clean up any existing timer
	tm.stopTimer()

	tm.sessionDuration = duration
	tm.currentState = state
	tm.timeRemaining = duration
	tm.startTime = time.Now()
	tm.onTick = onTick
	tm.onComplete = onComplete

	// Create context for cancellation
	tm.ctx, tm.cancel = context.WithCancel(context.Background())

	// Start the main timer
	tm.timer = time.AfterFunc(duration, func() {
		tm.handleSessionComplete()
	})

	// Start the ticker for periodic updates (every 100ms for more responsive updates)
	tm.ticker = time.NewTicker(100 * time.Millisecond)
	go tm.tickerLoop()
}

// PauseTimer pauses the timer and calculates remaining time
func (tm *TimerManager) PauseTimer() time.Duration {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.timer == nil {
		return tm.timeRemaining
	}

	// Calculate remaining time before stopping
	tm.timeRemaining = tm.getTimeRemainingLocked()
	tm.pauseTime = time.Now()

	// Stop the timer and ticker
	tm.stopTimer()

	log.Printf("⏸️ Timer paused with %v remaining", tm.timeRemaining)
	return tm.timeRemaining
}

// ResumeTimer resumes the timer from where it was paused
func (tm *TimerManager) ResumeTimer() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.timeRemaining <= 0 {
		log.Printf("⚠️ Cannot resume timer: no time remaining")
		return
	}

	// Ensure any existing timer is stopped first
	tm.stopTimer()

	// Create new context
	tm.ctx, tm.cancel = context.WithCancel(context.Background())

	// Update start time to account for the pause duration
	if !tm.pauseTime.IsZero() {
		pauseDuration := time.Since(tm.pauseTime)
		tm.startTime = time.Now().Add(-pauseDuration)
	} else {
		// If no pause time recorded, just use current time as start
		tm.startTime = time.Now()
	}

	// Update session duration to match remaining time
	tm.sessionDuration = tm.timeRemaining

	// Start the timer with remaining time
	tm.timer = time.AfterFunc(tm.timeRemaining, func() {
		tm.handleSessionComplete()
	})

	// Start the ticker
	tm.ticker = time.NewTicker(100 * time.Millisecond)
	go tm.tickerLoop()

	log.Printf("▶️ Timer resumed with %v remaining", tm.timeRemaining)
}

// StopTimer stops the timer completely
func (tm *TimerManager) StopTimer() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.stopTimer()
}

// GetTimeRemaining returns the current remaining time
func (tm *TimerManager) GetTimeRemaining() time.Duration {
	// No lock needed for reading - simple data access
	if tm.timer == nil {
		return tm.timeRemaining
	}

	// Calculate remaining time based on elapsed time
	elapsed := time.Since(tm.startTime)
	if elapsed >= tm.sessionDuration {
		return 0
	}
	return tm.sessionDuration - elapsed
}

// getTimeRemainingLocked calculates remaining time with lock held (internal method)
func (tm *TimerManager) getTimeRemainingLocked() time.Duration {
	if tm.timer == nil {
		return tm.timeRemaining
	}

	// Calculate remaining time based on elapsed time
	elapsed := time.Since(tm.startTime)
	if elapsed >= tm.sessionDuration {
		return 0
	}
	return tm.sessionDuration - elapsed
}

// IsRunning returns true if the timer is currently running
func (tm *TimerManager) IsRunning() bool {
	// No lock needed - in polling scenarios, slightly stale reads are acceptable
	return tm.timer != nil
}

// stopTimer is an internal method to stop the timer and ticker
func (tm *TimerManager) stopTimer() {
	if tm.timer != nil {
		tm.timer.Stop()
		tm.timer = nil
	}

	if tm.ticker != nil {
		tm.ticker.Stop()
		tm.ticker = nil
	}

	if tm.cancel != nil {
		tm.cancel()
		tm.cancel = nil
	}
}

// tickerLoop runs the ticker loop for periodic updates
func (tm *TimerManager) tickerLoop() {
	ticker := tm.ticker // Capture ticker reference
	if ticker == nil {
		// This can happen if ticker is stopped between check and goroutine start
		return
	}

	for {
		select {
		case <-ticker.C:
			if tm.onTick != nil && tm.timer != nil {
				remaining := tm.GetTimeRemaining()
				tm.onTick(remaining)
			}
		case <-tm.ctx.Done():
			return
		}
	}
}

// handleSessionComplete handles session completion
func (tm *TimerManager) handleSessionComplete() {
	tm.mu.Lock()

	// Stop the ticker
	if tm.ticker != nil {
		tm.ticker.Stop()
		tm.ticker = nil
	}

	// Call completion callback while still holding the lock
	if tm.onComplete != nil {
		completionState := tm.currentState
		tm.mu.Unlock() // Temporarily release lock for callback
		tm.onComplete(completionState)
		tm.mu.Lock() // Re-acquire lock
	}

	// Reset timer state after completion logic has run
	tm.timer = nil
	tm.timeRemaining = 0

	tm.mu.Unlock()
}
