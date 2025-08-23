package clock

import (
	"sync"
	"time"
)

// StatisticsManager handles tracking of pomodoro session statistics
type StatisticsManager struct {
	mu sync.RWMutex

	// Session completion counts
	totalWorkSessions int
	totalShortBreaks  int
	totalLongBreaks   int

	// Timing statistics
	totalWorkTime    time.Duration
	totalBreakTime   time.Duration
	totalSessionTime time.Duration

	// Session history
	sessionHistory []SessionRecord
}

// SessionRecord represents a completed session
type SessionRecord struct {
	State     ClockState
	Duration  time.Duration
	Completed time.Time
}

// NewStatisticsManager creates a new statistics manager
func NewStatisticsManager() *StatisticsManager {
	return &StatisticsManager{
		sessionHistory: make([]SessionRecord, 0),
	}
}

// RecordSession records a completed session
func (sm *StatisticsManager) RecordSession(state ClockState, duration time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	record := SessionRecord{
		State:     state,
		Duration:  duration,
		Completed: time.Now(),
	}

	sm.sessionHistory = append(sm.sessionHistory, record)

	switch state {
	case StateWorking:
		sm.totalWorkSessions++
		sm.totalWorkTime += duration
	case StateShortBreak:
		sm.totalShortBreaks++
		sm.totalBreakTime += duration
	case StateLongBreak:
		sm.totalLongBreaks++
		sm.totalBreakTime += duration
	}

	sm.totalSessionTime += duration
}

// GetStatistics returns the current statistics
func (sm *StatisticsManager) GetStatistics() (workSessions, shortBreaks, longBreaks int) {
	// No lock needed - in polling scenarios, slightly stale reads are acceptable
	// and write operations should have priority
	return sm.totalWorkSessions, sm.totalShortBreaks, sm.totalLongBreaks
}

// GetTimingStatistics returns timing statistics
func (sm *StatisticsManager) GetTimingStatistics() (workTime, breakTime, totalTime time.Duration) {
	// No lock needed - in polling scenarios, slightly stale reads are acceptable
	// and write operations should have priority
	return sm.totalWorkTime, sm.totalBreakTime, sm.totalSessionTime
}

// GetSessionHistory returns the session history
func (sm *StatisticsManager) GetSessionHistory() []SessionRecord {
	// No lock needed for reading - may return slightly stale data during writes
	history := make([]SessionRecord, len(sm.sessionHistory))
	copy(history, sm.sessionHistory)
	return history
}

// GetRecentSessions returns the most recent N sessions
func (sm *StatisticsManager) GetRecentSessions(count int) []SessionRecord {
	// No lock needed for reading - may return slightly stale data during writes
	if count <= 0 {
		return []SessionRecord{}
	}

	if count > len(sm.sessionHistory) {
		count = len(sm.sessionHistory)
	}

	start := len(sm.sessionHistory) - count
	recent := make([]SessionRecord, count)
	copy(recent, sm.sessionHistory[start:])
	return recent
}

// GetSessionsByState returns all sessions of a specific state
func (sm *StatisticsManager) GetSessionsByState(state ClockState) []SessionRecord {
	// No lock needed for reading - may return slightly stale data during writes
	var sessions []SessionRecord
	for _, record := range sm.sessionHistory {
		if record.State == state {
			sessions = append(sessions, record)
		}
	}
	return sessions
}

// GetTodaySessions returns sessions completed today
func (sm *StatisticsManager) GetTodaySessions() []SessionRecord {
	// No lock needed for reading - may return slightly stale data during writes
	today := time.Now().Truncate(24 * time.Hour)
	var todaySessions []SessionRecord

	for _, record := range sm.sessionHistory {
		if record.Completed.After(today) {
			todaySessions = append(todaySessions, record)
		}
	}

	return todaySessions
}

// GetWeeklyStats returns statistics for the current week
func (sm *StatisticsManager) GetWeeklyStats() (workSessions, shortBreaks, longBreaks int, workTime, breakTime time.Duration) {
	// No lock needed for reading - may return slightly stale data during writes
	weekStart := time.Now().Truncate(24 * time.Hour)
	weekStart = weekStart.AddDate(0, 0, -int(weekStart.Weekday()))

	for _, record := range sm.sessionHistory {
		if record.Completed.After(weekStart) {
			switch record.State {
			case StateWorking:
				workSessions++
				workTime += record.Duration
			case StateShortBreak:
				shortBreaks++
				breakTime += record.Duration
			case StateLongBreak:
				longBreaks++
				breakTime += record.Duration
			}
		}
	}

	return
}

// ResetStatistics resets all statistics
func (sm *StatisticsManager) ResetStatistics() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.totalWorkSessions = 0
	sm.totalShortBreaks = 0
	sm.totalLongBreaks = 0
	sm.totalWorkTime = 0
	sm.totalBreakTime = 0
	sm.totalSessionTime = 0
	sm.sessionHistory = make([]SessionRecord, 0)
}

// GetAverageSessionDuration returns the average duration of completed sessions
func (sm *StatisticsManager) GetAverageSessionDuration() time.Duration {
	// No lock needed - in polling scenarios, slightly stale reads are acceptable
	// and write operations should have priority
	totalSessions := sm.totalWorkSessions + sm.totalShortBreaks + sm.totalLongBreaks
	if totalSessions == 0 {
		return 0
	}

	return sm.totalSessionTime / time.Duration(totalSessions)
}

// GetProductivityScore returns a simple productivity score based on work sessions
func (sm *StatisticsManager) GetProductivityScore() float64 {
	// No lock needed for reading simple integer values
	totalSessions := sm.totalWorkSessions + sm.totalShortBreaks + sm.totalLongBreaks
	if totalSessions == 0 {
		return 0.0
	}

	// Productivity score = (work sessions / total sessions) * 100
	return float64(sm.totalWorkSessions) / float64(totalSessions) * 100.0
}
