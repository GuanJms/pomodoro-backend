package test

import (
	"context"
	"log"
	"testing"
	"time"

	"pomodoroService/internal/clock"

	"github.com/redis/go-redis/v9"
)

func TestRedisIntegration(t *testing.T) {
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

	// Test Redis persistence creation
	redisPersistence, err := clock.NewRedisPersistence(redisAddr)
	if err != nil {
		t.Fatalf("Failed to create Redis persistence: %v", err)
	}
	defer redisPersistence.Close()

	// Test settings save/load
	settings := &clock.PomodoroSettings{
		WorkTime:       30,
		ShortBreakTime: 10,
		LongBreakTime:  20,
		Scheduling:     "custom",
	}

	err = redisPersistence.SaveSettings(settings)
	if err != nil {
		t.Fatalf("Failed to save settings: %v", err)
	}

	loadedSettings, err := redisPersistence.LoadSettings()
	if err != nil {
		t.Fatalf("Failed to load settings: %v", err)
	}

	if loadedSettings.WorkTime != settings.WorkTime {
		t.Errorf("Expected work time %d, got %d", settings.WorkTime, loadedSettings.WorkTime)
	}

	// Test system state save/load
	loc := time.Now().Location()
	timezone := loc.String()
	state := &clock.SystemState{
		CurrentSession: 2,
		EndTime:        time.Now().Add(10 * time.Minute),
		Timezone:       timezone,
		State:          string(clock.StateWorking),
		TimeRemaining:  600,
		IsRunning:      true,
		IsPaused:       false,
	}

	err = redisPersistence.SaveSystemState(state)
	if err != nil {
		t.Fatalf("Failed to save system state: %v", err)
	}

	loadedState, err := redisPersistence.LoadSystemState()
	if err != nil {
		t.Fatalf("Failed to load system state: %v", err)
	}

	if loadedState.CurrentSession != state.CurrentSession {
		t.Errorf("Expected current session %d, got %d", state.CurrentSession, loadedState.CurrentSession)
	}

	if loadedState.State != state.State {
		t.Errorf("Expected state %s, got %s", state.State, loadedState.State)
	}

	log.Println("Redis integration test passed")
}

func TestClockRunnerWithRedis(t *testing.T) {
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

	// Test clock runner with Redis
	cr, err := clock.NewClockRunnerWithRedis(redisAddr)
	if err != nil {
		t.Fatalf("Failed to create clock runner with Redis: %v", err)
	}
	defer cr.Close()

	// Set custom durations
	cr.SetDurations(30*time.Minute, 10*time.Minute, 20*time.Minute)

	// Verify settings were saved to Redis
	redisPersistence, err := clock.NewRedisPersistence(redisAddr)
	if err != nil {
		t.Fatalf("Failed to create Redis persistence: %v", err)
	}
	defer redisPersistence.Close()

	settings, err := redisPersistence.LoadSettings()
	if err != nil {
		t.Fatalf("Failed to load settings: %v", err)
	}

	if settings.WorkTime != 30 {
		t.Errorf("Expected work time 30, got %d", settings.WorkTime)
	}

	log.Println("Clock runner with Redis test passed")
}
