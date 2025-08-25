package clock

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisPersistence handles Redis operations for clock state and settings
type RedisPersistence struct {
	client *redis.Client
	ctx    context.Context
}

// PomodoroSettings represents the settings stored in Redis
type PomodoroSettings struct {
	WorkTime       int    `json:"workTime"`
	ShortBreakTime int    `json:"shortBreakTime"`
	LongBreakTime  int    `json:"longBreakTime"`
	Scheduling     string `json:"scheduling"`
}

// SystemState represents the current system state stored in Redis
type SystemState struct {
	CurrentSession int       `json:"currentSession"`
	EndTime        time.Time `json:"endTime"`
	Timezone       string    `json:"timezone"`
	State          string    `json:"state"`
	TimeRemaining  int64     `json:"timeRemaining"` // in milliseconds for better precision
	IsRunning      bool      `json:"isRunning"`
	IsPaused       bool      `json:"isPaused"`
}

// NewRedisPersistence creates a new Redis persistence instance
func NewRedisPersistence(addr string) (*RedisPersistence, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	ctx := context.Background()

	// Test the connection
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("✅ Connected to Redis at %s", addr)
	return &RedisPersistence{
		client: client,
		ctx:    ctx,
	}, nil
}

// Close closes the Redis connection
func (rp *RedisPersistence) Close() error {
	return rp.client.Close()
}

// SaveSettings saves pomodoro settings to Redis
func (rp *RedisPersistence) SaveSettings(settings *PomodoroSettings) error {
	err := rp.client.HSet(rp.ctx, "pomodoroSettings", map[string]interface{}{
		"workTime":       settings.WorkTime,
		"shortBreakTime": settings.ShortBreakTime,
		"longBreakTime":  settings.LongBreakTime,
		"scheduling":     settings.Scheduling,
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to save settings to Redis: %w", err)
	}

	log.Printf("Saved pomodoro settings to Redis")
	return nil
}

// LoadSettings loads pomodoro settings from Redis
func (rp *RedisPersistence) LoadSettings() (*PomodoroSettings, error) {
	result, err := rp.client.HGetAll(rp.ctx, "pomodoroSettings").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load settings from Redis: %w", err)
	}

	if len(result) == 0 {
		// Return default settings if none exist
		return &PomodoroSettings{
			WorkTime:       25,
			ShortBreakTime: 5,
			LongBreakTime:  15,
			Scheduling:     "default",
		}, nil
	}

	settings := &PomodoroSettings{}

	// Parse the values (Redis returns strings)
	if workTime, ok := result["workTime"]; ok {
		if _, err := fmt.Sscanf(workTime, "%d", &settings.WorkTime); err != nil {
			settings.WorkTime = 25 // default
		}
	}

	if shortBreakTime, ok := result["shortBreakTime"]; ok {
		if _, err := fmt.Sscanf(shortBreakTime, "%d", &settings.ShortBreakTime); err != nil {
			settings.ShortBreakTime = 5 // default
		}
	}

	if longBreakTime, ok := result["longBreakTime"]; ok {
		if _, err := fmt.Sscanf(longBreakTime, "%d", &settings.LongBreakTime); err != nil {
			settings.LongBreakTime = 15 // default
		}
	}

	if scheduling, ok := result["scheduling"]; ok {
		settings.Scheduling = scheduling
	} else {
		settings.Scheduling = "default"
	}

	return settings, nil
}

// SaveSystemState saves the current system state to Redis
func (rp *RedisPersistence) SaveSystemState(state *SystemState) error {
	err := rp.client.HSet(rp.ctx, "systemState", map[string]interface{}{
		"currentSession": state.CurrentSession,
		"endTime":        state.EndTime.Format(time.RFC3339),
		"timezone":       state.Timezone,
		"state":          state.State,
		"timeRemaining":  state.TimeRemaining,
		"isRunning":      state.IsRunning,
		"isPaused":       state.IsPaused,
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to save system state to Redis: %w", err)
	}

	log.Printf("Saved system state to Redis: session=%d, state=%s, remaining=%dms",
		state.CurrentSession, state.State, state.TimeRemaining)
	return nil
}

// getSystemTimezone returns the current system timezone
func (rp *RedisPersistence) getSystemTimezone() string {
	zone, _ := time.Now().Zone()
	return zone
}

// getSystemTimezoneName returns the full timezone name (e.g., "America/New_York")
func (rp *RedisPersistence) getSystemTimezoneName() string {
	loc := time.Now().Location()
	return loc.String()
}

// LoadSystemState loads the system state from Redis
func (rp *RedisPersistence) LoadSystemState() (*SystemState, error) {
	result, err := rp.client.HGetAll(rp.ctx, "systemState").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load system state from Redis: %w", err)
	}

	if len(result) == 0 {
		// Return default state if none exist
		return &SystemState{
			CurrentSession: 0,
			EndTime:        time.Now().Add(24 * time.Hour), // Set to 24 hours from now for idle state
			Timezone:       rp.getSystemTimezoneName(),
			State:          string(StateIdle),
			TimeRemaining:  0,
			IsRunning:      false,
			IsPaused:       false,
		}, nil
	}

	state := &SystemState{}

	// Parse current session
	if currentSession, ok := result["currentSession"]; ok {
		if _, err := fmt.Sscanf(currentSession, "%d", &state.CurrentSession); err != nil {
			state.CurrentSession = 0
		}
	}

	// Parse end time
	if endTime, ok := result["endTime"]; ok {
		if parsedTime, err := time.Parse(time.RFC3339, endTime); err == nil {
			state.EndTime = parsedTime
		} else {
			state.EndTime = time.Now()
		}
	}

	// Parse timezone
	if timezone, ok := result["timezone"]; ok {
		state.Timezone = timezone
	} else {
		state.Timezone = rp.getSystemTimezoneName()
	}

	// Parse state
	if stateStr, ok := result["state"]; ok {
		state.State = stateStr
	} else {
		state.State = string(StateIdle)
	}

	// Parse time remaining
	if timeRemaining, ok := result["timeRemaining"]; ok {
		if _, err := fmt.Sscanf(timeRemaining, "%d", &state.TimeRemaining); err != nil {
			log.Printf("⚠️ Invalid time remaining in Redis, defaulting to 0: %v", err)
			state.TimeRemaining = 0
		}
	}

	// Parse isRunning
	if isRunning, ok := result["isRunning"]; ok {
		state.IsRunning = (isRunning == "true" || isRunning == "1")
	}

	// Parse isPaused
	if isPaused, ok := result["isPaused"]; ok {
		state.IsPaused = (isPaused == "true" || isPaused == "1")
	}

	// Validate and repair the loaded state
	rp.validateAndRepairState(state)

	return state, nil
}

// validateAndRepairState validates loaded state and repairs inconsistencies
func (rp *RedisPersistence) validateAndRepairState(state *SystemState) {
	repairs := 0

	// Validate session range (assuming max 100 sessions for safety)
	if state.CurrentSession < 0 || state.CurrentSession > 100 {
		log.Printf("⚠️ Invalid session number %d, resetting to 0", state.CurrentSession)
		state.CurrentSession = 0
		repairs++
	}

	// Validate state string
	validStates := map[string]bool{
		string(StateIdle):       true,
		string(StateWorking):    true,
		string(StateShortBreak): true,
		string(StateLongBreak):  true,
		string(StatePaused):     true,
	}
	if !validStates[state.State] {
		log.Printf("⚠️ Invalid state '%s', resetting to 'idle'", state.State)
		state.State = string(StateIdle)
		repairs++
	}

	// Validate state consistency
	if state.State == string(StateIdle) && (state.IsRunning || state.IsPaused || state.TimeRemaining > 0) {
		log.Printf("⚠️ Idle state has running flags, fixing")
		state.IsRunning = false
		state.IsPaused = false
		state.TimeRemaining = 0
		repairs++
	}

	if state.IsRunning && state.IsPaused {
		log.Printf("⚠️ Cannot be both running and paused, setting to paused")
		state.IsRunning = false
		repairs++
	}

	if state.TimeRemaining < 0 {
		log.Printf("⚠️ Negative time remaining %d, setting to 0", state.TimeRemaining)
		state.TimeRemaining = 0
		repairs++
	}

	if repairs > 0 {
		log.Printf("🔧 Repaired %d inconsistencies in Redis state", repairs)
	}
}

// SaveSessionStatistics saves session statistics to Redis
func (rp *RedisPersistence) SaveSessionStatistics(sessionType string, duration time.Duration, completedAt time.Time) error {
	key := fmt.Sprintf("session_stats:%s:%s", sessionType, completedAt.Format("2006-01-02"))

	sessionData := map[string]interface{}{
		"type":        sessionType,
		"duration":    int(duration.Seconds()),
		"completedAt": completedAt.Format(time.RFC3339),
	}

	err := rp.client.HSet(rp.ctx, key, sessionData).Err()
	if err != nil {
		return fmt.Errorf("failed to save session statistics: %w", err)
	}

	// Set expiration for 30 days
	rp.client.Expire(rp.ctx, key, 30*24*time.Hour)

	log.Printf("Saved session statistics: type=%s, duration=%v", sessionType, duration)
	return nil
}
