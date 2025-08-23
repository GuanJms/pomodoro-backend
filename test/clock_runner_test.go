package test

import (
	"sync"
	"testing"
	"time"

	. "pomodoroService/internal/clock"
)

func TestNewClockRunner(t *testing.T) {
	cr := NewClockRunner()

	if cr == nil {
		t.Fatal("NewClockRunner returned nil")
	}

	if cr.GetState() != StateIdle {
		t.Errorf("Expected initial state to be StateIdle, got %s", cr.GetState())
	}

	if cr.GetCurrentSession() != 0 {
		t.Errorf("Expected initial session to be 0, got %d", cr.GetCurrentSession())
	}

	if cr.GetTotalSessions() != 8 {
		t.Errorf("Expected total sessions to be 8, got %d", cr.GetTotalSessions())
	}

	// Test that durations are set correctly by checking session manager
	// Note: We can't directly access durations anymore, but we can verify through behavior
}

func TestSetDurations(t *testing.T) {
	cr := NewClockRunner()

	workDur := 30 * time.Minute
	shortDur := 10 * time.Minute
	longDur := 20 * time.Minute

	cr.SetDurations(workDur, shortDur, longDur)

	// Test that durations are set correctly by checking session manager behavior
	// We can verify this by starting a session and checking the duration
	cr.Start()
	remaining := cr.GetTimeRemaining()
	if remaining <= 0 || remaining > workDur {
		t.Errorf("Expected remaining time to be between 0 and %v, got %v", workDur, remaining)
	}
	cr.Stop()
}

func TestStart(t *testing.T) {
	cr := NewClockRunner()
	cr.SetDurations(1*time.Second, 500*time.Millisecond, 1*time.Second)

	// Test starting from idle state
	err := cr.Start()
	if err != nil {
		t.Errorf("Expected no error when starting from idle, got %v", err)
	}

	if cr.GetState() != StateWorking {
		t.Errorf("Expected state to be StateWorking after start, got %s", cr.GetState())
	}

	if !cr.IsRunning() {
		t.Error("Expected clock to be running after start")
	}

	// Test starting when already running
	err = cr.Start()
	if err == nil {
		t.Error("Expected error when starting already running clock")
	}
}

func TestPause(t *testing.T) {
	cr := NewClockRunner()
	cr.SetDurations(1*time.Second, 500*time.Millisecond, 1*time.Second)

	// Start the clock
	cr.Start()

	// Wait a bit to ensure it's running
	time.Sleep(100 * time.Millisecond)

	// Test pausing
	err := cr.Pause()
	if err != nil {
		t.Errorf("Expected no error when pausing, got %v", err)
	}

	if cr.GetState() != StatePaused {
		t.Errorf("Expected state to be StatePaused after pause, got %s", cr.GetState())
	}

	if !cr.IsPaused() {
		t.Error("Expected clock to be paused")
	}

	// Test pausing when already paused
	err = cr.Pause()
	if err == nil {
		t.Error("Expected error when pausing already paused clock")
	}
}

func TestResumeFromPause(t *testing.T) {
	cr := NewClockRunner()
	cr.SetDurations(1*time.Second, 500*time.Millisecond, 1*time.Second)

	// Start and pause
	cr.Start()
	time.Sleep(100 * time.Millisecond)
	cr.Pause()

	// Get remaining time before resume
	remainingBefore := cr.GetTimeRemaining()

	// Resume
	err := cr.Start()
	if err != nil {
		t.Errorf("Expected no error when resuming, got %v", err)
	}

	if cr.GetState() != StateWorking {
		t.Errorf("Expected state to be StateWorking after resume, got %s", cr.GetState())
	}

	// Check that remaining time is preserved
	remainingAfter := cr.GetTimeRemaining()
	if remainingAfter > remainingBefore {
		t.Errorf("Expected remaining time to be preserved or decreased, got %v > %v", remainingAfter, remainingBefore)
	}
}

func TestStop(t *testing.T) {
	cr := NewClockRunner()
	cr.SetDurations(1*time.Second, 500*time.Millisecond, 1*time.Second)

	// Start the clock
	cr.Start()
	time.Sleep(100 * time.Millisecond)

	// Test stopping
	err := cr.Stop()
	if err != nil {
		t.Errorf("Expected no error when stopping, got %v", err)
	}

	if cr.GetState() != StateIdle {
		t.Errorf("Expected state to be StateIdle after stop, got %s", cr.GetState())
	}

	if !cr.IsIdle() {
		t.Error("Expected clock to be idle after stop")
	}

	if cr.GetTimeRemaining() != 0 {
		t.Errorf("Expected remaining time to be 0 after stop, got %v", cr.GetTimeRemaining())
	}

	// Test stopping when already idle
	err = cr.Stop()
	if err == nil {
		t.Error("Expected error when stopping already idle clock")
	}
}

func TestSkip(t *testing.T) {
	cr := NewClockRunner()
	cr.SetDurations(1*time.Second, 500*time.Millisecond, 1*time.Second)

	// Start the clock
	cr.Start()

	// Test skipping
	err := cr.Skip()
	if err != nil {
		t.Errorf("Expected no error when skipping, got %v", err)
	}

	// Should move to short break
	if cr.GetState() != StateShortBreak {
		t.Errorf("Expected state to be StateShortBreak after skip, got %s", cr.GetState())
	}

	if cr.GetCurrentSession() != 1 {
		t.Errorf("Expected session to be 1 after skip, got %d", cr.GetCurrentSession())
	}

	// Test skipping when idle
	cr.Stop()
	err = cr.Skip()
	if err == nil {
		t.Error("Expected error when skipping idle clock")
	}
}

func TestGetTimeRemaining(t *testing.T) {
	cr := NewClockRunner()

	// Set short durations for testing
	cr.SetDurations(2*time.Second, 1*time.Second, 3*time.Second)

	// Test when idle
	remaining := cr.GetTimeRemaining()
	if remaining != 0 {
		t.Errorf("Expected remaining time to be 0 when idle, got %v", remaining)
	}

	// Test when running
	cr.Start()
	time.Sleep(500 * time.Millisecond)

	remaining = cr.GetTimeRemaining()
	if remaining <= 0 || remaining > 2*time.Second {
		t.Errorf("Expected remaining time to be between 0 and 2 seconds, got %v", remaining)
	}

	// Test when paused
	cr.Pause()
	pausedRemaining := cr.GetTimeRemaining()
	time.Sleep(100 * time.Millisecond)

	// Remaining time should not change when paused
	if cr.GetTimeRemaining() != pausedRemaining {
		t.Errorf("Expected remaining time to be unchanged when paused, got %v != %v", cr.GetTimeRemaining(), pausedRemaining)
	}
}

func TestGetFormattedTimeRemaining(t *testing.T) {
	cr := NewClockRunner()

	// Test when idle
	formatted := cr.GetFormattedTimeRemaining()
	if formatted != "00:00" {
		t.Errorf("Expected formatted time to be '00:00' when idle, got %s", formatted)
	}

	// Set short duration and start
	cr.SetDurations(65*time.Second, 1*time.Second, 3*time.Second)
	cr.Start()

	formatted = cr.GetFormattedTimeRemaining()
	// Should be "01:05" or close to it
	if formatted != "01:05" && formatted != "01:04" {
		t.Errorf("Expected formatted time to be around '01:05', got %s", formatted)
	}
}

func TestCallbacks(t *testing.T) {
	cr := NewClockRunner()

	var stateChanges []ClockState
	var ticks []time.Duration
	var completions []ClockState

	// Set callbacks
	cr.SetCallbacks(
		func(state ClockState) {
			stateChanges = append(stateChanges, state)
		},
		func(remaining time.Duration) {
			ticks = append(ticks, remaining)
		},
		func(completedState ClockState) {
			completions = append(completions, completedState)
		},
	)

	// Set longer duration for testing to ensure ticks are recorded
	cr.SetDurations(2*time.Second, 1*time.Second, 1*time.Second)

	// Start the clock
	cr.Start()

	// Wait for a few ticks to occur
	time.Sleep(3 * time.Second)

	// Stop the clock
	cr.Stop()

	// Check state changes
	if len(stateChanges) == 0 {
		t.Error("Expected state changes to be recorded")
	}

	// Check ticks (should have at least 1 tick for a 2-second session)
	if len(ticks) < 1 {
		t.Errorf("Expected at least 1 tick to be recorded, got %d", len(ticks))
	}

	// Check completions
	if len(completions) == 0 {
		t.Error("Expected completions to be recorded")
	}
}

func TestClockRunnerStatistics(t *testing.T) {
	cr := NewClockRunner()

	// Set short durations
	cr.SetDurations(50*time.Millisecond, 25*time.Millisecond, 75*time.Millisecond)

	// Start and let it run through a few sessions
	cr.Start()
	time.Sleep(200 * time.Millisecond)

	workSessions, shortBreaks, _ := cr.GetStatistics()

	// Should have at least one work session
	if workSessions == 0 {
		t.Error("Expected at least one work session to be recorded")
	}

	// Should have some short breaks
	if shortBreaks == 0 {
		t.Error("Expected at least one short break to be recorded")
	}
}

func TestClockRunnerConcurrentAccess(t *testing.T) {
	cr := NewClockRunner()
	cr.SetDurations(1*time.Second, 500*time.Millisecond, 1*time.Second)

	var wg sync.WaitGroup
	numGoroutines := 10

	// Test concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				cr.GetState()
				cr.GetTimeRemaining()
				cr.GetCurrentSession()
				cr.GetTotalSessions()
				cr.IsRunning()
				cr.IsPaused()
				cr.IsIdle()
			}
		}()
	}

	wg.Wait()

	// Test concurrent writes (should be safe due to mutex)
	cr.Start()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cr.Pause()
			cr.Start()
		}()
	}

	wg.Wait()
}

func TestSessionProgression(t *testing.T) {
	cr := NewClockRunner()
	cr.SetDurations(200*time.Millisecond, 100*time.Millisecond, 150*time.Millisecond)

	// Start the clock
	cr.Start()

	// Verify initial state
	if cr.GetState() != StateWorking {
		t.Errorf("Expected initial state to be StateWorking, got %s", cr.GetState())
	}

	// Wait for first session to complete
	time.Sleep(300 * time.Millisecond)

	// Should be in short break
	if cr.GetState() != StateShortBreak {
		t.Errorf("Expected state to be StateShortBreak after work session, got %s", cr.GetState())
	}

	// Wait for short break to complete
	time.Sleep(150 * time.Millisecond)

	// Should be back to work
	if cr.GetState() != StateWorking {
		t.Errorf("Expected state to be StateWorking after short break, got %s", cr.GetState())
	}
}

func TestClockRunnerCompleteCycle(t *testing.T) {
	cr := NewClockRunner()
	cr.SetDurations(20*time.Millisecond, 10*time.Millisecond, 15*time.Millisecond)

	// Start the clock
	cr.Start()

	// Wait for complete cycle (8 sessions)
	time.Sleep(500 * time.Millisecond)

	// Should be back to idle
	if cr.GetState() != StateIdle {
		t.Errorf("Expected state to be StateIdle after complete cycle, got %s", cr.GetState())
	}

	if cr.GetCurrentSession() != 0 {
		t.Errorf("Expected session to be reset to 0 after complete cycle, got %d", cr.GetCurrentSession())
	}
}

func TestContextCancellation(t *testing.T) {
	cr := NewClockRunner()
	cr.SetDurations(1*time.Second, 500*time.Millisecond, 1*time.Second)

	// Start the clock
	cr.Start()

	// Stop the clock (this will cancel the context internally)
	cr.Stop()

	// Should still be able to access state
	state := cr.GetState()
	if state != StateIdle {
		t.Errorf("Expected state to be StateIdle after stop, got %s", state)
	}
}

func BenchmarkClockRunner(b *testing.B) {
	cr := NewClockRunner()
	cr.SetDurations(1*time.Second, 500*time.Millisecond, 1*time.Second)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cr.Start()
		cr.Pause()
		cr.Start()
		cr.Stop()
	}
}

func BenchmarkGetTimeRemaining(b *testing.B) {
	cr := NewClockRunner()
	cr.SetDurations(1*time.Second, 500*time.Millisecond, 1*time.Second)
	cr.Start()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cr.GetTimeRemaining()
	}
}
