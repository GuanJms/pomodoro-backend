package main

import (
	"encoding/json"
	"net/http"
	"pomodoroService/internal/clock"
	"time"
)

func NewClockHandler(clockRunner *clock.ClockRunner) *ClockHandler {
	return &ClockHandler{clockRunner: clockRunner}
}

type ClockHandler struct {
	clockRunner *clock.ClockRunner
}

// SystemStateResponse represents the system state response format
type SystemStateResponse struct {
	PomodoroSetting struct {
		WorkTimeDuration   int    `json:"workTimeDuration"`
		LongBreakDuration  int    `json:"longBreakDuration"`
		ShortBreakDuration int    `json:"shortBreakDuration"`
		Scheduling         string `json:"scheduling"`
	} `json:"pomodoroSetting"`
	CurrentSession int    `json:"currentSession"`
	EndTime        string `json:"endTime"`
	ServerTime     string `json:"serverTime"`
}

func (h *ClockHandler) GetSystemState(w http.ResponseWriter, r *http.Request) {
	// Get current time
	now := time.Now()

	// Get schedule and format it
	schedule := h.clockRunner.GetSchedule()
	scheduling := clock.FormatScheduling(schedule)

	// Load state from Redis once for consistency
	var redisState *clock.SystemState
	var redisError error
	redisPersistence := h.clockRunner.GetRedisPersistence()
	if redisPersistence != nil {
		redisState, redisError = redisPersistence.LoadSystemState()
	}

	// Get current session directly from Redis for consistency
	currentSession := 0
	if redisError == nil && redisState != nil {
		currentSession = redisState.CurrentSession
		// log.Printf("📊 API using Redis currentSession: %d", currentSession)
	} else {
		// Fallback to in-memory value if Redis fails
		currentSession = h.clockRunner.GetCurrentSession()
		// log.Printf("📊 API using in-memory currentSession: %d (Redis error: %v)", currentSession, redisError)
	}

	// Get the end time from Redis directly to avoid recalculation inconsistencies
	var endTime time.Time
	if h.clockRunner.IsIdle() {
		// For idle state, use 24 hours from now
		endTime = now.Add(24 * time.Hour)
		// log.Printf("📊 API using calculated endTime for idle: %s", endTime.Format(time.RFC3339))
	} else {
		// For active states, get the exact end time from Redis
		if redisError == nil && redisState != nil && !redisState.EndTime.IsZero() {
			endTime = redisState.EndTime
			// log.Printf("📊 API using Redis endTime: %s", endTime.Format(time.RFC3339))
		} else {
			// Fallback to calculation if Redis load fails
			endTime = now.Add(h.clockRunner.GetTimeRemaining())
			// log.Printf("📊 API using calculated endTime (Redis error: %v): %s", redisError, endTime.Format(time.RFC3339))
		}
	}

	// Create response
	response := SystemStateResponse{}

	// Set pomodoro settings
	workDuration, shortBreakDuration, longBreakDuration := h.clockRunner.GetDurations()
	response.PomodoroSetting.WorkTimeDuration = workDuration
	response.PomodoroSetting.LongBreakDuration = longBreakDuration
	response.PomodoroSetting.ShortBreakDuration = shortBreakDuration
	response.PomodoroSetting.Scheduling = scheduling

	// Set session info
	response.CurrentSession = currentSession

	// Set times
	response.EndTime = endTime.Format(time.RFC3339)
	response.ServerTime = now.Format(time.RFC3339)

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Encode and send response
	json.NewEncoder(w).Encode(response)
}

func (h *ClockHandler) StartNewPomodoro(w http.ResponseWriter, r *http.Request) {
	err := h.clockRunner.Start()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Pomodoro started"))
}
