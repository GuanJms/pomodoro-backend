package clock

import (
	"fmt"
	"log"
	"time"
)

// PersistenceManager handles Redis persistence operations for the clock runner
type PersistenceManager struct {
	clockRunner *ClockRunner
}

// NewPersistenceManager creates a new persistence manager
func NewPersistenceManager(clockRunner *ClockRunner) *PersistenceManager {
	return &PersistenceManager{
		clockRunner: clockRunner,
	}
}

// LoadSettingsFromRedis loads settings from Redis and applies them
func (pm *PersistenceManager) LoadSettingsFromRedis() error {
	if pm.clockRunner.redisPersistence == nil {
		return fmt.Errorf("redis persistence not initialized")
	}

	settings, err := pm.clockRunner.redisPersistence.LoadSettings()
	if err != nil {
		return err
	}

	// Apply the loaded settings
	pm.clockRunner.SetDurations(
		time.Duration(settings.WorkTime)*time.Minute,
		time.Duration(settings.ShortBreakTime)*time.Minute,
		time.Duration(settings.LongBreakTime)*time.Minute,
	)

	return nil
}

// SaveSettingsToRedis saves current settings to Redis
func (pm *PersistenceManager) SaveSettingsToRedis() error {
	if pm.clockRunner.redisPersistence == nil {
		return fmt.Errorf("redis persistence not initialized")
	}

	workMinutes, shortBreakMinutes, longBreakMinutes := pm.clockRunner.GetDurations()
	settings := &PomodoroSettings{
		WorkTime:       int(workMinutes),
		ShortBreakTime: int(shortBreakMinutes),
		LongBreakTime:  int(longBreakMinutes),
		Scheduling:     "default", // TODO: get from session manager
	}

	return pm.clockRunner.redisPersistence.SaveSettings(settings)
}

// SaveSystemStateToRedis saves the current system state to Redis
func (pm *PersistenceManager) SaveSystemStateToRedis() error {
	if pm.clockRunner.redisPersistence == nil {
		return fmt.Errorf("redis persistence not initialized")
	}

	// Get current state
	currentState := pm.clockRunner.GetState()
	timeRemaining := pm.clockRunner.GetTimeRemaining()

	// Calculate end time based on current state
	var endTime time.Time
	var timeRemainingSeconds int64

	if currentState == StateIdle {
		// For idle state, set end time to 1 hour from now and no remaining time
		endTime = time.Now().Add(1 * time.Hour)
		timeRemainingSeconds = 0
	} else {
		// For active states, calculate precise end time and remaining time
		if timeRemaining > 0 {
			endTime = time.Now().Add(timeRemaining)
			// Use higher precision by storing microseconds
			timeRemainingSeconds = int64(timeRemaining.Nanoseconds() / 1000000) // Convert to milliseconds for better precision
		} else {
			endTime = time.Now()
			timeRemainingSeconds = 0
		}
	}

	// Get system timezone
	timezone := time.Now().Location().String()

	// Get running/paused status
	isRunning := pm.clockRunner.IsRunning()
	isPaused := pm.clockRunner.IsPaused()

	state := &SystemState{
		CurrentSession: pm.clockRunner.GetCurrentSession(),
		EndTime:        endTime,
		Timezone:       timezone,
		State:          string(currentState),
		TimeRemaining:  timeRemainingSeconds,
		IsRunning:      isRunning,
		IsPaused:       isPaused,
	}

	log.Printf("💾 Saving to Redis - Session: %d, State: %s, Remaining: %dms, Running: %v, Paused: %v, EndTime: %s",
		state.CurrentSession, state.State, state.TimeRemaining, state.IsRunning, state.IsPaused,
		state.EndTime.Format("15:04:05"))

	log.Printf("Checking state manager is running: %v", pm.clockRunner.stateManager.IsRunning())
	log.Printf("Checking if timer is running: %v", pm.clockRunner.timerManager.IsRunning())
	return pm.clockRunner.redisPersistence.SaveSystemState(state)
}

// SaveSessionStatistics saves session statistics to Redis
func (pm *PersistenceManager) SaveSessionStatistics(sessionType string, duration time.Duration) error {
	if pm.clockRunner.redisPersistence == nil {
		return fmt.Errorf("redis persistence not initialized")
	}

	return pm.clockRunner.redisPersistence.SaveSessionStatistics(sessionType, duration, time.Now())
}

// Close closes the Redis connection
func (pm *PersistenceManager) Close() error {
	if pm.clockRunner.redisPersistence != nil {
		return pm.clockRunner.redisPersistence.Close()
	}
	return nil
}
