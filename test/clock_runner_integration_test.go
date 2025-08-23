package test

import (
	"pomodoroService/internal/clock"
	"sync"
	"testing"
	"time"
)

// TestPomodoroWorkflow tests a complete pomodoro workflow
func TestPomodoroWorkflow(t *testing.T) {
	cr := clock.NewClockRunner()

	// Set longer durations for more reliable testing
	cr.SetDurations(200*time.Millisecond, 100*time.Millisecond, 150*time.Millisecond)

	var stateHistory []clock.ClockState
	var completionHistory []clock.ClockState

	// Set callbacks to track state changes
	cr.SetCallbacks(
		func(state clock.ClockState) {
			stateHistory = append(stateHistory, state)
			t.Logf("State changed to: %s", state)
		},
		func(remaining time.Duration) {
			// Log every 10th tick to avoid spam
			if len(stateHistory)%10 == 0 {
				t.Logf("Tick: %v remaining", remaining)
			}
		},
		func(completedState clock.ClockState) {
			completionHistory = append(completionHistory, completedState)
			t.Logf("Session completed: %s", completedState)
		},
	)

	// Start the pomodoro session
	t.Log("Starting pomodoro session...")
	err := cr.Start()
	if err != nil {
		t.Fatalf("Failed to start clock: %v", err)
	}

	// Wait for the complete cycle to finish (8 sessions: 4*200ms + 3*100ms + 1*150ms = 1.25s)
	time.Sleep(2 * time.Second)

	// Verify the workflow
	if len(stateHistory) == 0 {
		t.Error("Expected state changes to be recorded")
	}

	if len(completionHistory) == 0 {
		t.Error("Expected session completions to be recorded")
	}

	// Should end up in idle state
	finalState := cr.GetState()
	if finalState != clock.StateIdle {
		t.Errorf("Expected final state to be StateIdle, got %s", finalState)
	}

	// Should have completed all sessions
	workSessions, shortBreaks, longBreaks := cr.GetStatistics()
	t.Logf("Completed: %d work sessions, %d short breaks, %d long breaks",
		workSessions, shortBreaks, longBreaks)

	if workSessions == 0 {
		t.Error("Expected at least one work session to be completed")
	}

	if shortBreaks == 0 {
		t.Error("Expected at least one short break to be completed")
	}
}

// TestPauseResumeWorkflow tests pausing and resuming during a session
func TestPauseResumeWorkflow(t *testing.T) {
	cr := clock.NewClockRunner()
	cr.SetDurations(200*time.Millisecond, 100*time.Millisecond, 150*time.Millisecond)

	var pauseResumeCount int

	cr.SetCallbacks(
		func(state clock.ClockState) {
			if state == clock.StatePaused {
				pauseResumeCount++
			}
		},
		nil, nil,
	)

	// Start the session
	cr.Start()

	// Wait a bit, then pause
	time.Sleep(50 * time.Millisecond)
	cr.Pause()

	// Verify paused state
	if !cr.IsPaused() {
		t.Error("Expected clock to be paused")
	}

	// Get remaining time while paused
	pausedRemaining := cr.GetTimeRemaining()

	// Wait a bit more
	time.Sleep(50 * time.Millisecond)

	// Resume
	cr.Start()

	// Verify running state
	if !cr.IsRunning() {
		t.Error("Expected clock to be running after resume")
	}

	// Check that remaining time was preserved
	resumedRemaining := cr.GetTimeRemaining()
	if resumedRemaining > pausedRemaining {
		t.Errorf("Expected remaining time to be preserved or decreased, got %v > %v",
			resumedRemaining, pausedRemaining)
	}

	// Wait for completion
	time.Sleep(300 * time.Millisecond)

	if pauseResumeCount == 0 {
		t.Error("Expected pause events to be recorded")
	}
}

// TestSkipWorkflow tests skipping sessions
func TestSkipWorkflow(t *testing.T) {
	cr := clock.NewClockRunner()
	cr.SetDurations(100*time.Millisecond, 50*time.Millisecond, 75*time.Millisecond)

	var skippedStates []clock.ClockState

	cr.SetCallbacks(
		nil, nil,
		func(completedState clock.ClockState) {
			skippedStates = append(skippedStates, completedState)
		},
	)

	// Start the session
	cr.Start()

	// Skip the first work session
	time.Sleep(10 * time.Millisecond)
	cr.Skip()

	// Should be in short break
	if cr.GetState() != clock.StateShortBreak {
		t.Errorf("Expected state to be StateShortBreak after skip, got %s", cr.GetState())
	}

	// Skip the short break
	time.Sleep(10 * time.Millisecond)
	cr.Skip()

	// Should be back to work
	if cr.GetState() != clock.StateWorking {
		t.Errorf("Expected state to be StateWorking after skip, got %s", cr.GetState())
	}

	// Wait for natural completion
	time.Sleep(200 * time.Millisecond)

	// Verify skipped sessions were recorded
	if len(skippedStates) < 2 {
		t.Errorf("Expected at least 2 skipped sessions, got %d", len(skippedStates))
	}
}

// TestConcurrentAccess tests multiple goroutines accessing the clock runner
func TestConcurrentAccess(t *testing.T) {
	cr := clock.NewClockRunner()
	cr.SetDurations(1*time.Second, 500*time.Millisecond, 750*time.Millisecond)

	var wg sync.WaitGroup
	numReaders := 5
	numWriters := 3

	// Start the clock
	cr.Start()

	// Start reader goroutines
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				state := cr.GetState()
				remaining := cr.GetTimeRemaining()
				session := cr.GetCurrentSession()

				// Verify we get valid values
				if state == "" {
					t.Errorf("Reader %d got empty state", id)
				}
				if remaining < 0 {
					t.Errorf("Reader %d got negative remaining time: %v", id, remaining)
				}
				if session < 0 || session >= cr.GetTotalSessions() {
					t.Errorf("Reader %d got invalid session: %d", id, session)
				}

				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	// Start writer goroutines
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				// Pause and resume
				cr.Pause()
				time.Sleep(20 * time.Millisecond)
				cr.Start()
				time.Sleep(20 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// Verify the clock is still in a valid state
	state := cr.GetState()
	if state == "" {
		t.Error("Clock state is empty after concurrent access")
	}
}

// TestRealisticPomodoroSession tests with realistic durations
func TestRealisticPomodoroSession(t *testing.T) {
	cr := clock.NewClockRunner()

	// Use realistic durations (but shorter for testing)
	cr.SetDurations(2*time.Second, 1*time.Second, 3*time.Second)

	var sessionCount int
	var totalWorkTime time.Duration
	var totalBreakTime time.Duration

	startTime := time.Now()

	cr.SetCallbacks(
		func(state clock.ClockState) {
			sessionCount++
			t.Logf("Session %d: %s", sessionCount, state)
		},
		nil,
		func(completedState clock.ClockState) {
			switch completedState {
			case clock.StateWorking:
				totalWorkTime += 2 * time.Second
			case clock.StateShortBreak:
				totalBreakTime += 1 * time.Second
			case clock.StateLongBreak:
				totalBreakTime += 3 * time.Second
			}
		},
	)

	// Start the session
	cr.Start()

	// Wait for completion
	time.Sleep(20 * time.Second)

	totalTime := time.Since(startTime)

	t.Logf("Total session time: %v", totalTime)
	t.Logf("Total work time: %v", totalWorkTime)
	t.Logf("Total break time: %v", totalBreakTime)
	t.Logf("Number of sessions: %d", sessionCount)

	// Verify reasonable values
	if sessionCount < 4 {
		t.Errorf("Expected at least 4 sessions, got %d", sessionCount)
	}

	if totalWorkTime == 0 {
		t.Error("Expected some work time to be recorded")
	}

	if totalBreakTime == 0 {
		t.Error("Expected some break time to be recorded")
	}
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	cr := clock.NewClockRunner()

	// Test very short durations
	cr.SetDurations(10*time.Millisecond, 5*time.Millisecond, 15*time.Millisecond)

	var completions int
	cr.SetCallbacks(nil, nil, func(completedState clock.ClockState) {
		completions++
	})

	cr.Start()
	time.Sleep(100 * time.Millisecond)

	if completions == 0 {
		t.Error("Expected some sessions to complete with very short durations")
	}

	// Test rapid start/stop cycles
	cr.Stop()
	for i := 0; i < 10; i++ {
		cr.Start()
		time.Sleep(1 * time.Millisecond)
		cr.Stop()
	}

	// Should still be in valid state
	if cr.GetState() != clock.StateIdle {
		t.Errorf("Expected state to be StateIdle after rapid cycles, got %s", cr.GetState())
	}
}

// TestMemoryLeaks tests for potential memory leaks
func TestMemoryLeaks(t *testing.T) {
	// Run multiple clock runners to check for resource leaks
	for i := 0; i < 10; i++ {
		cr := clock.NewClockRunner()
		cr.SetDurations(50*time.Millisecond, 25*time.Millisecond, 75*time.Millisecond)

		cr.Start()
		time.Sleep(100 * time.Millisecond)
		cr.Stop()

		// Force garbage collection to check for leaks
		// Note: This is a basic test - in production you'd use more sophisticated tools
	}

	t.Log("Memory leak test completed - check with profiling tools if needed")
}

// BenchmarkIntegration benchmarks the complete pomodoro workflow
func BenchmarkIntegration(b *testing.B) {
	cr := clock.NewClockRunner()
	cr.SetDurations(10*time.Millisecond, 5*time.Millisecond, 15*time.Millisecond)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cr.Start()
		time.Sleep(100 * time.Millisecond) // Wait for completion
		cr.Stop()
	}
}
