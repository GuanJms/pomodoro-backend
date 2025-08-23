package clock

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// ClockState represents the current state of the pomodoro clock
type ClockState string

const (
	StateIdle       ClockState = "I"
	StateWorking    ClockState = "W"
	StateShortBreak ClockState = "SB"
	StateLongBreak  ClockState = "LB"
	StatePaused     ClockState = "P"
)

var ClockStateMap = map[string]ClockState{
	"I":  StateIdle,
	"W":  StateWorking,
	"SB": StateShortBreak,
	"LB": StateLongBreak,
	"P":  StatePaused,
}

// ClockRunner manages the pomodoro timer and state using modular components
type ClockRunner struct {
	mu sync.RWMutex

	// Component managers
	stateManager   *StateManager
	sessionManager *SessionManager
	timerManager   *TimerManager
	statsManager   *StatisticsManager
	timeFormatter  *TimeFormatter
	utils          *ClockUtils

	// Redis persistence
	redisPersistence *RedisPersistence

	// Manager components
	persistenceManager *PersistenceManager
	resumeManager      *ResumeManager

	// Tick counter for Redis persistence
	tickCounter int

	// Callbacks
	onStateChange func(ClockState)
	onTick        func(time.Duration)
	onComplete    func(ClockState)
}

// NewClockRunner creates a new clock runner with default settings
func NewClockRunner() *ClockRunner {
	return &ClockRunner{
		stateManager:   NewStateManager(),
		sessionManager: NewSessionManager(),
		timerManager:   NewTimerManager(),
		statsManager:   NewStatisticsManager(),
		timeFormatter:  NewTimeFormatter(),
		utils:          NewClockUtils(),
	}
}

// NewClockRunnerWithRedis creates a new clock runner with Redis persistence
func NewClockRunnerWithRedis(redisAddr string) (*ClockRunner, error) {
	redisPersistence, err := NewRedisPersistence(redisAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Redis persistence: %w", err)
	}

	cr := &ClockRunner{
		stateManager:     NewStateManager(),
		sessionManager:   NewSessionManager(),
		timerManager:     NewTimerManager(),
		statsManager:     NewStatisticsManager(),
		timeFormatter:    NewTimeFormatter(),
		utils:            NewClockUtils(),
		redisPersistence: redisPersistence,
	}

	// Initialize manager components
	cr.persistenceManager = NewPersistenceManager(cr)
	cr.resumeManager = NewResumeManager(cr)

	// Load settings from Redis
	if err := cr.persistenceManager.LoadSettingsFromRedis(); err != nil {
		log.Printf("Warning: failed to load settings from Redis: %v", err)
	}

	// Load and resume system state from Redis
	if err := cr.resumeManager.ResumeFromRedis(); err != nil {
		log.Printf("Warning: failed to resume state from Redis: %v", err)
	}

	return cr, nil
}

// SetDurations configures the clock runner with custom durations
func (cr *ClockRunner) SetDurations(work, shortBreak, longBreak time.Duration) {
	cr.sessionManager.SetDurations(work, shortBreak, longBreak)

	// Save settings to Redis
	if cr.redisPersistence != nil {
		if err := cr.persistenceManager.SaveSettingsToRedis(); err != nil {
			log.Printf("Failed to save settings to Redis: %v", err)
		}
	}
}

// SetCallbacks sets the callback functions for state changes and ticks
func (cr *ClockRunner) SetCallbacks(
	onStateChange func(ClockState),
	onTick func(time.Duration),
	onComplete func(ClockState),
) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Store callbacks directly
	cr.onStateChange = onStateChange
	cr.onTick = onTick
	cr.onComplete = onComplete
}

// Start begins the pomodoro session
func (cr *ClockRunner) Start() error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	log.Printf("🚀 Start() called - Current state: %s, CanStart: %v", cr.GetState(), cr.stateManager.CanStart())

	if !cr.stateManager.CanStart() {
		return fmt.Errorf("cannot start: clock is not in a startable state")
	}

	if cr.stateManager.IsIdle() {
		log.Printf("Starting new session from idle state")
		cr.sessionManager.ResetSessions()
		cr.startNewSession()
	} else if cr.stateManager.IsPaused() {
		log.Printf("Resuming from paused state")
		// Resume from pause
		state := cr.sessionManager.GetCurrentSessionState()
		cr.stateManager.SetState(state)
		cr.timerManager.ResumeTimer()
	}

	// Save state to Redis
	cr.saveStateToRedis()

	log.Printf("✅ Start() completed - Final state: %s, Session: %d", cr.GetState(), cr.GetCurrentSession())
	return nil
}

// Pause pauses the current session
func (cr *ClockRunner) Pause() error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if !cr.stateManager.CanPause() {
		return fmt.Errorf("cannot pause: clock is not running")
	}

	cr.stateManager.SetState(StatePaused)
	cr.timerManager.PauseTimer()

	// Save state to Redis
	cr.saveStateToRedis()

	if cr.onStateChange != nil {
		cr.onStateChange(StatePaused)
	}

	return nil
}

// Stop stops the current session and resets to idle
func (cr *ClockRunner) Stop() error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if !cr.stateManager.CanStop() {
		return fmt.Errorf("cannot stop: clock is already idle")
	}

	cr.stateManager.SetState(StateIdle)
	cr.sessionManager.ResetSessions()
	cr.timerManager.StopTimer()

	// Save state to Redis
	if cr.redisPersistence != nil {
		cr.saveStateToRedis()
	}

	if cr.onStateChange != nil {
		cr.onStateChange(StateIdle)
	}

	return nil
}

// Skip skips the current session and moves to the next one
func (cr *ClockRunner) Skip() error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if !cr.stateManager.CanSkip() {
		return fmt.Errorf("cannot skip: clock is not running")
	}

	// Stop the current timer
	cr.timerManager.StopTimer()

	// Call completion callback
	if cr.onComplete != nil {
		cr.onComplete(cr.stateManager.GetState())
	}

	// Record the skipped session
	duration := cr.sessionManager.GetCurrentSessionDuration()
	cr.statsManager.RecordSession(cr.stateManager.GetState(), duration)

	log.Printf("Skipped %s session %d/%d",
		cr.stateManager.GetState(), cr.sessionManager.GetCurrentSession(), cr.sessionManager.GetTotalSessions())

	// Move to next session
	if !cr.sessionManager.NextSession() {
		// Completed all sessions, reset
		cr.stateManager.SetState(StateIdle)

		// Save state to Redis
		cr.saveStateToRedis()

		if cr.onStateChange != nil {
			cr.onStateChange(StateIdle)
		}
		log.Println("Completed all pomodoro sessions!")
		return nil
	}

	cr.startNewSession()
	return nil
}

// startNewSession starts a new session
func (cr *ClockRunner) startNewSession() {
	state := cr.sessionManager.GetCurrentSessionState()
	duration := cr.sessionManager.GetCurrentSessionDuration()

	cr.stateManager.SetState(state)

	// Reset tick counter for new session
	cr.tickCounter = 0

	// Set up timer callbacks
	onTick := func(remaining time.Duration) {
		if cr.onTick != nil {
			cr.onTick(remaining)
		}
	}

	onComplete := func(completedState ClockState) {
		// Record the completed session
		cr.statsManager.RecordSession(completedState, duration)

		if cr.onComplete != nil {
			cr.onComplete(completedState)
		}

		log.Printf("Completed %s session %d/%d",
			completedState, cr.sessionManager.GetCurrentSession(), cr.sessionManager.GetTotalSessions())

		// Move to next session
		log.Printf("Moving to next session from %d", cr.sessionManager.GetCurrentSession())
		hasNextSession := cr.sessionManager.NextSession()
		log.Printf("Next session available: %v, current session now: %d", hasNextSession, cr.sessionManager.GetCurrentSession())
		if !hasNextSession {
			// Completed all sessions - set to idle and save state
			cr.stateManager.SetState(StateIdle)
			if cr.onStateChange != nil {
				cr.onStateChange(StateIdle)
			}
			// Save idle state to Redis immediately
			cr.saveStateToRedis()
			log.Println("Completed all pomodoro sessions!")
			return
		}

		// Start next session - this will save state to Redis
		cr.startNewSession()

		// Ensure the new session state is saved to Redis immediately
		cr.saveStateToRedis()
	}

	// Start the timer
	cr.timerManager.StartTimer(duration, state, onTick, onComplete)

	// Log session start
	state, sessionNum, totalSessions, _ := cr.sessionManager.GetSessionInfo()
	log.Printf("Started %s session %d/%d (duration: %v)",
		state, sessionNum, totalSessions, duration)

	if cr.onStateChange != nil {
		cr.onStateChange(state)
	}

	// Save state to Redis
	cr.saveStateToRedis()
}

// Close closes the Redis connection
func (cr *ClockRunner) Close() error {
	if cr.persistenceManager != nil {
		return cr.persistenceManager.Close()
	}
	return nil
}

// GetState returns the current state
func (cr *ClockRunner) GetState() ClockState {
	return cr.stateManager.GetState()
}

// GetCurrentSession returns the current session number (0-based)
func (cr *ClockRunner) GetCurrentSession() int {
	return cr.sessionManager.GetCurrentSession()
}

// GetTotalSessions returns the total number of sessions
func (cr *ClockRunner) GetTotalSessions() int {
	return cr.sessionManager.GetTotalSessions()
}

// GetTimeRemaining returns the remaining time for the current session
func (cr *ClockRunner) GetTimeRemaining() time.Duration {
	return cr.timerManager.GetTimeRemaining()
}

// GetFormattedTimeRemaining returns the formatted remaining time
func (cr *ClockRunner) GetFormattedTimeRemaining() string {
	remaining := cr.GetTimeRemaining()
	return cr.timeFormatter.FormatDuration(remaining)
}

// IsRunning returns true if the clock is running
func (cr *ClockRunner) IsRunning() bool {
	// Check both state and timer status - no lock needed for reading
	stateRunning := cr.stateManager.IsRunning()
	timerRunning := cr.timerManager.IsRunning()

	// Log inconsistency for debugging
	if stateRunning != timerRunning {
		log.Printf("⚠️ State/timer inconsistency: state running=%v, timer running=%v", stateRunning, timerRunning)
	}

	return stateRunning && timerRunning
}

// IsPaused returns true if the clock is paused
func (cr *ClockRunner) IsPaused() bool {
	return cr.stateManager.IsPaused()
}

// IsIdle returns true if the clock is idle
func (cr *ClockRunner) IsIdle() bool {
	return cr.stateManager.IsIdle()
}

// GetStatistics returns the session statistics
func (cr *ClockRunner) GetStatistics() (int, int, int) {
	return cr.statsManager.GetStatistics()
}

// GetTimingStatistics returns timing statistics
func (cr *ClockRunner) GetTimingStatistics() (time.Duration, time.Duration, time.Duration) {
	return cr.statsManager.GetTimingStatistics()
}

// GetSessionHistory returns the session history
func (cr *ClockRunner) GetSessionHistory() []SessionRecord {
	return cr.statsManager.GetSessionHistory()
}

// GetRecentSessions returns the most recent N sessions
func (cr *ClockRunner) GetRecentSessions(count int) []SessionRecord {
	return cr.statsManager.GetRecentSessions(count)
}

// GetTodaySessions returns sessions completed today
func (cr *ClockRunner) GetTodaySessions() []SessionRecord {
	return cr.statsManager.GetTodaySessions()
}

// GetWeeklyStats returns statistics for the current week
func (cr *ClockRunner) GetWeeklyStats() (int, int, int, time.Duration, time.Duration) {
	return cr.statsManager.GetWeeklyStats()
}

// GetAverageSessionDuration returns the average duration of completed sessions
func (cr *ClockRunner) GetAverageSessionDuration() time.Duration {
	return cr.statsManager.GetAverageSessionDuration()
}

// GetProductivityScore returns a simple productivity score
func (cr *ClockRunner) GetProductivityScore() float64 {
	return cr.statsManager.GetProductivityScore()
}

// ResetStatistics resets all statistics
func (cr *ClockRunner) ResetStatistics() {
	cr.statsManager.ResetStatistics()
}

// SetSchedule sets a custom session schedule
func (cr *ClockRunner) SetSchedule(schedule []ClockState) error {
	return cr.sessionManager.SetSchedule(schedule)
}

// GetSchedule returns the current schedule
func (cr *ClockRunner) GetSchedule() []ClockState {
	return cr.sessionManager.GetSchedule()
}

// GetDurations returns the current durations in minutes
func (cr *ClockRunner) GetDurations() (workMinutes, shortBreakMinutes, longBreakMinutes int) {
	return cr.sessionManager.GetDurations()
}

// GetScheduleSummary returns a summary of the schedule
func (cr *ClockRunner) GetScheduleSummary() map[ClockState]int {
	return cr.utils.GetScheduleSummary(cr.sessionManager.GetSchedule())
}

// GetRedisPersistence returns the Redis persistence instance
func (cr *ClockRunner) GetRedisPersistence() *RedisPersistence {
	return cr.redisPersistence
}

// GetSessionManager returns the session manager instance
func (cr *ClockRunner) GetSessionManager() *SessionManager {
	return cr.sessionManager
}

// synchronizeStateTimer ensures state and timer are consistent
func (cr *ClockRunner) synchronizeStateTimer() {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	stateRunning := cr.stateManager.IsRunning()
	timerRunning := cr.timerManager.IsRunning()

	// If state says running but timer is not running, fix the state
	if stateRunning && !timerRunning && cr.stateManager.GetState() != StatePaused {
		log.Printf("🔄 Fixing state/timer inconsistency: timer stopped, updating state to idle")
		cr.stateManager.SetState(StateIdle)
		if cr.onStateChange != nil {
			cr.onStateChange(StateIdle)
		}
	}

	// If timer is running but state says idle/paused, fix the timer
	if !stateRunning && timerRunning {
		log.Printf("🔄 Fixing state/timer inconsistency: stopping orphaned timer")
		cr.timerManager.StopTimer()
	}
}

// saveStateToRedis saves the current state to Redis with tick-based frequency control
func (cr *ClockRunner) saveStateToRedis() {
	if cr.redisPersistence == nil {
		return
	}

	// Save state to Redis (less frequently for ticks)
	if cr.onTick != nil {
		cr.tickCounter++
		if cr.tickCounter >= 10 { // Every 10 ticks (1 second with 100ms intervals)
			if err := cr.persistenceManager.SaveSystemStateToRedis(); err != nil {
				log.Printf("Failed to save state to Redis: %v", err)
			}
			cr.tickCounter = 0
		}
	} else {
		// For state changes, always save immediately
		if err := cr.persistenceManager.SaveSystemStateToRedis(); err != nil {
			log.Printf("Failed to save state to Redis: %v", err)
		}
	}
}
