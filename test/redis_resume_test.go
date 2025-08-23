package test

import (
	"context"
	"log"
	"testing"
	"time"

	"pomodoroService/internal/clock"

	"github.com/redis/go-redis/v9"
)

const redisAddr = "localhost:6379"

func TestRedisResumeFunctionality(t *testing.T) {
	// Skip if Redis is not available
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer client.Close()

	// Test 1: Resume running session
	t.Run("ResumeRunningSession", func(t *testing.T) {
		redisPersistence, err := clock.NewRedisPersistence(redisAddr)
		if err != nil {
			t.Fatalf("Failed to create Redis persistence: %v", err)
		}
		defer redisPersistence.Close()

		// Save a running session state
		state := &clock.SystemState{
			CurrentSession: 1,
			EndTime:        time.Now().Add(10 * time.Minute), // 10 minutes from now
			Timezone:       time.Now().Location().String(),
			State:          string(clock.StateWorking),
			TimeRemaining:  600, // 10 minutes in seconds
			IsRunning:      true,
			IsPaused:       false,
		}

		err = redisPersistence.SaveSystemState(state)
		if err != nil {
			t.Fatalf("Failed to save state: %v", err)
		}

		// Create clock runner with Redis
		cr, err := clock.NewClockRunnerWithRedis(redisAddr)
		if err != nil {
			t.Fatalf("Failed to create clock runner: %v", err)
		}
		defer cr.Close()

		// Verify the state was resumed
		if cr.GetCurrentSession() != 1 {
			t.Errorf("Expected current session 1, got %d", cr.GetCurrentSession())
		}

		if cr.GetState() != clock.StateWorking {
			t.Errorf("Expected state working, got %s", cr.GetState())
		}

		if !cr.IsRunning() {
			t.Error("Expected clock to be running")
		}

		log.Println("Resume running session test passed")
	})

	// Test 2: Resume paused session
	t.Run("ResumePausedSession", func(t *testing.T) {
		redisPersistence, err := clock.NewRedisPersistence(redisAddr)
		if err != nil {
			t.Fatalf("Failed to create Redis persistence: %v", err)
		}
		defer redisPersistence.Close()

		// Save a paused session state
		state := &clock.SystemState{
			CurrentSession: 2,
			EndTime:        time.Now().Add(5 * time.Minute), // 5 minutes from now
			Timezone:       time.Now().Location().String(),
			State:          string(clock.StateShortBreak),
			TimeRemaining:  300, // 5 minutes in seconds
			IsRunning:      false,
			IsPaused:       true,
		}

		err = redisPersistence.SaveSystemState(state)
		if err != nil {
			t.Fatalf("Failed to save state: %v", err)
		}

		// Create clock runner with Redis
		cr, err := clock.NewClockRunnerWithRedis(redisAddr)
		if err != nil {
			t.Fatalf("Failed to create clock runner: %v", err)
		}
		defer cr.Close()

		// Verify the state was resumed
		if cr.GetCurrentSession() != 2 {
			t.Errorf("Expected current session 2, got %d", cr.GetCurrentSession())
		}

		if cr.GetState() != clock.StatePaused {
			t.Errorf("Expected state paused, got %s", cr.GetState())
		}

		if !cr.IsPaused() {
			t.Error("Expected clock to be paused")
		}

		log.Println("Resume paused session test passed")
	})

	// Test 3: Reset when server was offline too long
	t.Run("ResetWhenOfflineTooLong", func(t *testing.T) {
		redisPersistence, err := clock.NewRedisPersistence(redisAddr)
		if err != nil {
			t.Fatalf("Failed to create Redis persistence: %v", err)
		}
		defer redisPersistence.Close()

		// Save a state with end time in the past (server was offline too long)
		state := &clock.SystemState{
			CurrentSession: 1,
			EndTime:        time.Now().Add(-1 * time.Hour), // 1 hour ago
			Timezone:       time.Now().Location().String(),
			State:          string(clock.StateWorking),
			TimeRemaining:  600,
			IsRunning:      true,
			IsPaused:       false,
		}

		err = redisPersistence.SaveSystemState(state)
		if err != nil {
			t.Fatalf("Failed to save state: %v", err)
		}

		// Create clock runner with Redis
		cr, err := clock.NewClockRunnerWithRedis(redisAddr)
		if err != nil {
			t.Fatalf("Failed to create clock runner: %v", err)
		}
		defer cr.Close()

		// Verify the state was reset
		if cr.GetCurrentSession() != 0 {
			t.Errorf("Expected current session 0 (reset), got %d", cr.GetCurrentSession())
		}

		if cr.GetState() != clock.StateIdle {
			t.Errorf("Expected state idle (reset), got %s", cr.GetState())
		}

		if cr.IsRunning() {
			t.Error("Expected clock to be idle after reset")
		}

		log.Println("Reset when offline too long test passed")
	})

	// Test 4: Resume idle state
	t.Run("ResumeIdleState", func(t *testing.T) {
		redisPersistence, err := clock.NewRedisPersistence(redisAddr)
		if err != nil {
			t.Fatalf("Failed to create Redis persistence: %v", err)
		}
		defer redisPersistence.Close()

		// Save an idle state
		state := &clock.SystemState{
			CurrentSession: 0,
			EndTime:        time.Now().Add(1 * time.Hour), // 1 hour from now
			Timezone:       time.Now().Location().String(),
			State:          string(clock.StateIdle),
			TimeRemaining:  0,
			IsRunning:      false,
			IsPaused:       false,
		}

		err = redisPersistence.SaveSystemState(state)
		if err != nil {
			t.Fatalf("Failed to save state: %v", err)
		}

		// Create clock runner with Redis
		cr, err := clock.NewClockRunnerWithRedis(redisAddr)
		if err != nil {
			t.Fatalf("Failed to create clock runner: %v", err)
		}
		defer cr.Close()

		// Verify the state was resumed
		if cr.GetCurrentSession() != 0 {
			t.Errorf("Expected current session 0, got %d", cr.GetCurrentSession())
		}

		if cr.GetState() != clock.StateIdle {
			t.Errorf("Expected state idle, got %s", cr.GetState())
		}

		if cr.IsRunning() {
			t.Error("Expected clock to be idle")
		}

		log.Println("Resume idle state test passed")
	})
}

func TestSessionManagerSetCurrentSession(t *testing.T) {
	sm := clock.NewSessionManager()

	// Test setting valid session
	sm.SetCurrentSession(2)
	if sm.GetCurrentSession() != 2 {
		t.Errorf("Expected session 2, got %d", sm.GetCurrentSession())
	}

	// Test setting negative session (should clamp to 0)
	sm.SetCurrentSession(-1)
	if sm.GetCurrentSession() != 0 {
		t.Errorf("Expected session 0 (clamped), got %d", sm.GetCurrentSession())
	}

	// Test setting session beyond schedule length (should clamp to max)
	sm.SetCurrentSession(10)
	expectedMax := sm.GetTotalSessions() - 1
	if sm.GetCurrentSession() != expectedMax {
		t.Errorf("Expected session %d (clamped), got %d", expectedMax, sm.GetCurrentSession())
	}

	log.Println("SessionManager SetCurrentSession test passed")
}
