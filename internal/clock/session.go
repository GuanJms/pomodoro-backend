package clock

import (
	"fmt"
	"sync"
	"time"
)

// SessionManager handles pomodoro session scheduling and progression
type SessionManager struct {
	mu sync.RWMutex

	// Session configuration
	workDuration       time.Duration
	shortBreakDuration time.Duration
	longBreakDuration  time.Duration
	schedule           []ClockState

	// Current session info
	currentSession int
}

// NewSessionManager creates a new session manager with default settings
func NewSessionManager() *SessionManager {
	return &SessionManager{
		workDuration:       25 * time.Minute,
		shortBreakDuration: 5 * time.Minute,
		longBreakDuration:  15 * time.Minute,
		schedule:           []ClockState{StateWorking, StateShortBreak, StateWorking, StateShortBreak, StateWorking, StateShortBreak, StateWorking, StateLongBreak},
		currentSession:     0,
	}
}

// SetDurations configures the session manager with custom durations
func (sm *SessionManager) SetDurations(work, shortBreak, longBreak time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.workDuration = work
	sm.shortBreakDuration = shortBreak
	sm.longBreakDuration = longBreak
}

// GetDurations returns the current durations in minutes
func (sm *SessionManager) GetDurations() (workMinutes, shortBreakMinutes, longBreakMinutes int) {
	// No lock needed - durations are only modified by write operations
	return int(sm.workDuration.Minutes()), int(sm.shortBreakDuration.Minutes()), int(sm.longBreakDuration.Minutes())
}

// GetCurrentSession returns the current session number (0-based)
func (sm *SessionManager) GetCurrentSession() int {
	// No lock needed - currentSession is only modified by write operations
	// and read operations don't need to be synchronized for this simple value
	return sm.currentSession
}

// GetTotalSessions returns the total number of sessions in the schedule
func (sm *SessionManager) GetTotalSessions() int {
	// No lock needed - schedule length is immutable once set
	return len(sm.schedule)
}

// GetCurrentSessionState returns the state of the current session
func (sm *SessionManager) GetCurrentSessionState() ClockState {
	// No lock needed - schedule is immutable once set, currentSession is only modified by writes
	if sm.currentSession >= len(sm.schedule) {
		return StateIdle
	}
	return sm.schedule[sm.currentSession]
}

// GetCurrentSessionDuration returns the duration of the current session
func (sm *SessionManager) GetCurrentSessionDuration() time.Duration {
	// No lock needed - all data is either immutable or only modified by write operations
	if sm.currentSession >= len(sm.schedule) {
		return 0
	}

	state := sm.schedule[sm.currentSession]
	switch state {
	case StateWorking:
		return sm.workDuration
	case StateShortBreak:
		return sm.shortBreakDuration
	case StateLongBreak:
		return sm.longBreakDuration
	default:
		return 0
	}
}

// NextSession advances to the next session
func (sm *SessionManager) NextSession() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.currentSession++
	if sm.currentSession >= len(sm.schedule) {
		// Completed all sessions, reset
		sm.currentSession = 0
		return false // Indicates cycle completion
	}
	return true // Indicates more sessions available
}

// ResetSessions resets the session counter to the beginning
func (sm *SessionManager) ResetSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.currentSession = 0
}

// SetCurrentSession sets the current session number (for resuming from persistence)
func (sm *SessionManager) SetCurrentSession(session int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Ensure session is within valid range
	if session < 0 {
		session = 0
	} else if session >= len(sm.schedule) {
		session = len(sm.schedule) - 1
	}

	sm.currentSession = session
}

// IsLastSession returns true if this is the last session in the cycle
func (sm *SessionManager) IsLastSession() bool {
	// No lock needed - schedule length is immutable, currentSession only modified by writes
	return sm.currentSession == len(sm.schedule)-1
}

// GetSessionInfo returns information about the current session
func (sm *SessionManager) GetSessionInfo() (state ClockState, sessionNum int, totalSessions int, duration time.Duration) {
	// No lock needed - all data is either immutable or only modified by write operations
	state = sm.GetCurrentSessionState()
	sessionNum = sm.currentSession // 0-based for consistency
	totalSessions = len(sm.schedule)
	duration = sm.GetCurrentSessionDuration()

	return
}

// GetSessionInfoAt returns information about a specific session
func (sm *SessionManager) GetSessionInfoAt(session int) (state ClockState, sessionNum int, totalSessions int, duration time.Duration) {
	// No lock needed - all data is either immutable or only modified by write operations
	if session < 0 || session >= len(sm.schedule) {
		return StateIdle, 0, len(sm.schedule), 0
	}

	state = sm.schedule[session]
	sessionNum = session
	totalSessions = len(sm.schedule)

	// Calculate duration for the specific session
	switch state {
	case StateWorking:
		duration = sm.workDuration
	case StateShortBreak:
		duration = sm.shortBreakDuration
	case StateLongBreak:
		duration = sm.longBreakDuration
	default:
		duration = 0
	}

	return
}

// GetSchedule returns a copy of the current schedule
func (sm *SessionManager) GetSchedule() []ClockState {
	// No lock needed for reading - simple data access
	schedule := make([]ClockState, len(sm.schedule))
	copy(schedule, sm.schedule)
	return schedule
}

// SetSchedule sets a custom session schedule
func (sm *SessionManager) SetSchedule(schedule []ClockState) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Validate schedule
	for _, state := range schedule {
		if state != StateWorking && state != StateShortBreak && state != StateLongBreak {
			return fmt.Errorf("invalid state in schedule: %s", state)
		}
	}

	if len(schedule) == 0 {
		return fmt.Errorf("schedule cannot be empty")
	}

	sm.schedule = make([]ClockState, len(schedule))
	copy(sm.schedule, schedule)
	sm.currentSession = 0 // Reset to beginning

	return nil
}
