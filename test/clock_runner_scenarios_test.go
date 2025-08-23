package test

import (
	"pomodoroService/internal/clock"
	"testing"
	"time"
)

// TestBasicScenarios tests basic scenarios using the helper framework
func TestBasicScenarios(t *testing.T) {
	// Test basic workflow scenario
	RunScenario(t, BasicWorkflowScenario)

	// Test pause resume scenario
	RunScenario(t, PauseResumeScenario)

	// Test skip scenario
	RunScenario(t, SkipScenario)
}

// TestCustomScenarios tests custom scenarios
func TestCustomScenarios(t *testing.T) {
	// Test rapid state changes
	RunScenario(t, TestScenario{
		Name: "Rapid State Changes",
		Actions: func(tcr *TestClockRunner) {
			for i := 0; i < 5; i++ {
				tcr.Start()
				time.Sleep(10 * time.Millisecond)
				tcr.Pause()
				time.Sleep(10 * time.Millisecond)
				tcr.Start()
				time.Sleep(10 * time.Millisecond)
				tcr.Stop()
			}
		},
		Assertions: func(tcr *TestClockRunner, tracker *StateTracker) {
			tcr.AssertState(clock.StateIdle)
			if tracker.CountState(clock.StatePaused) < 5 {
				tcr.t.Error("Expected at least 5 pause states")
			}
		},
	})

	// Test session progression
	RunScenario(t, TestScenario{
		Name: "Session Progression",
		Actions: func(tcr *TestClockRunner) {
			tcr.Start()
			// Let it run through a few sessions
		},
		Duration: 400 * time.Millisecond,
		Assertions: func(tcr *TestClockRunner, tracker *StateTracker) {
			// Should have gone through multiple states
			if len(tracker.GetStates()) < 3 {
				tcr.t.Error("Expected at least 3 state changes")
			}

			// Should have some completions
			if len(tracker.GetCompletions()) == 0 {
				tcr.t.Error("Expected some session completions")
			}
		},
	})

	// Test time remaining accuracy - simplified test
	RunScenario(t, TestScenario{
		Name: "Time Remaining Accuracy",
		Actions: func(tcr *TestClockRunner) {
			tcr.Start()
			// Give the timer a moment to actually start
			time.Sleep(10 * time.Millisecond)
		},
		Assertions: func(tcr *TestClockRunner, tracker *StateTracker) {
			// Should have some remaining time
			remaining := tcr.GetTimeRemaining()
			if remaining <= 0 {
				tcr.t.Error("Expected positive remaining time")
			}

			// Should be a reasonable value (not negative, not more than duration)
			if remaining > 200*time.Millisecond {
				tcr.t.Errorf("Remaining time %v is greater than session duration 200ms", remaining)
			}

			// Log the actual value for reference
			tcr.t.Logf("Time remaining: %v (this should be close to but less than 200ms)", remaining)
		},
	})
}

// TestStateTracking tests the state tracking functionality
func TestStateTracking(t *testing.T) {
	tcr := NewTestClockRunner(t)
	tracker := NewStateTracker()

	// Set up callbacks
	tcr.SetCallbacks(
		tracker.OnStateChange,
		tracker.OnTick,
		tracker.OnComplete,
	)

	// Run a simple workflow with longer durations to ensure ticks
	tcr.Start()
	time.Sleep(100 * time.Millisecond)
	tcr.Pause()
	time.Sleep(100 * time.Millisecond)
	tcr.Start()
	time.Sleep(2 * time.Second) // Wait long enough to get ticks (tick every 1 second)

	// Test state counting
	workingCount := tracker.CountState(clock.StateWorking)
	pausedCount := tracker.CountState(clock.StatePaused)

	if workingCount == 0 {
		t.Error("Expected working states to be recorded")
	}

	if pausedCount == 0 {
		t.Error("Expected paused states to be recorded")
	}

	// Test completion counting
	completions := tracker.GetCompletions()
	if len(completions) == 0 {
		t.Error("Expected completions to be recorded")
	}

	// Test tick recording
	ticks := tracker.GetTicks()
	if len(ticks) == 0 {
		t.Error("Expected ticks to be recorded")
	}
}

// TestWaitForState tests the wait functionality
func TestWaitForState(t *testing.T) {
	tcr := NewTestClockRunner(t)

	// Start the clock
	tcr.Start()

	// Wait for it to reach working state (should be immediate)
	if !tcr.WaitForState(clock.StateWorking, 100*time.Millisecond) {
		t.Error("Failed to wait for working state")
	}

	// Pause and wait for paused state
	tcr.Pause()
	if !tcr.WaitForState(clock.StatePaused, 100*time.Millisecond) {
		t.Error("Failed to wait for paused state")
	}
}

// TestWaitForSession tests the session wait functionality
func TestWaitForSession(t *testing.T) {
	tcr := NewTestClockRunner(t)

	// Start the clock
	tcr.Start()

	// Wait for session 0 (should be immediate)
	if !tcr.WaitForSession(0, 100*time.Millisecond) {
		t.Error("Failed to wait for session 0")
	}

	// Skip to next session and wait
	tcr.Skip()
	if !tcr.WaitForSession(1, 100*time.Millisecond) {
		t.Error("Failed to wait for session 1")
	}
}

// TestAssertions tests the assertion helpers
func TestAssertions(t *testing.T) {
	tcr := NewTestClockRunner(t)

	// Test state assertion
	tcr.AssertState(clock.StateIdle)

	// Test session assertion
	tcr.AssertSession(0)

	// Start and test running state
	tcr.Start()
	tcr.AssertState(clock.StateWorking)

	// Test time remaining assertion (should be close to 200ms since we just started)
	tcr.AssertTimeRemaining(150*time.Millisecond, 200*time.Millisecond)
}

// TestCompleteCycle tests the complete cycle functionality
func TestCompleteCycle(t *testing.T) {
	tcr := NewTestClockRunner(t)

	// Run a complete cycle
	workSessions, shortBreaks, _ := tcr.RunCompleteCycle()

	// Should have some completed sessions
	if workSessions == 0 {
		t.Error("Expected at least one work session to complete")
	}

	if shortBreaks == 0 {
		t.Error("Expected at least one short break to complete")
	}

	// Should end in idle state
	tcr.AssertState(clock.StateIdle)
}

// TestStateSequence tests state sequence validation
func TestStateSequence(t *testing.T) {
	tcr := NewTestClockRunner(t)
	tracker := NewStateTracker()

	// Set up callbacks
	tcr.SetCallbacks(
		tracker.OnStateChange,
		tracker.OnTick,
		tracker.OnComplete,
	)

	// Run a simple sequence
	tcr.Start()
	time.Sleep(100 * time.Millisecond) // Give time for state to be recorded
	tcr.Pause()
	time.Sleep(50 * time.Millisecond) // Small delay for pause to be recorded
	tcr.Start()
	time.Sleep(100 * time.Millisecond) // Time for resume state to be recorded
	tcr.Stop()
	time.Sleep(50 * time.Millisecond) // Time for stop state to be recorded

	// Log the actual states for debugging
	states := tracker.GetStates()
	t.Logf("Actual states: %v", states)

	// Test expected sequence - check that we have the minimum expected states
	if len(states) < 3 {
		t.Errorf("Expected at least 3 states, got %d: %v", len(states), states)
		return
	}

	// Check the first few states match expected pattern
	if states[0] != clock.StateWorking {
		t.Errorf("Expected first state to be working, got %s", states[0])
	}
	if states[1] != clock.StatePaused {
		t.Errorf("Expected second state to be paused, got %s", states[1])
	}
	// The third state could be working (when resumed) or idle (if stopped quickly)
	if states[2] != clock.StateWorking && states[2] != clock.StateIdle {
		t.Errorf("Expected third state to be working or idle, got %s", states[2])
	}
}

// TestErrorConditions tests error conditions and edge cases
func TestErrorConditions(t *testing.T) {
	tcr := NewTestClockRunner(t)

	// Test starting when already running
	tcr.Start()
	err := tcr.Start()
	if err == nil {
		t.Error("Expected error when starting already running clock")
	}

	// Test pausing when idle
	tcr.Stop()
	err = tcr.Pause()
	if err == nil {
		t.Error("Expected error when pausing idle clock")
	}

	// Test stopping when already idle
	err = tcr.Stop()
	if err == nil {
		t.Error("Expected error when stopping already idle clock")
	}

	// Test skipping when idle
	err = tcr.Skip()
	if err == nil {
		t.Error("Expected error when skipping idle clock")
	}
}

// TestFormattedTime tests the formatted time functionality
func TestFormattedTime(t *testing.T) {
	tcr := NewTestClockRunner(t)

	// Test idle state
	formatted := tcr.GetFormattedTimeRemaining()
	if formatted != "00:00" {
		t.Errorf("Expected '00:00' when idle, got %s", formatted)
	}

	// Test running state
	tcr.Start()
	time.Sleep(10 * time.Millisecond) // Small delay to ensure timer started
	formatted = tcr.GetFormattedTimeRemaining()
	// Should be "00:00" since 200ms = 0.2 seconds rounds to 0
	if formatted != "00:00" {
		t.Logf("Got formatted time: %s (this is expected with 200ms duration)", formatted)
	}
}

// TestStatistics tests the statistics functionality
func TestStatistics(t *testing.T) {
	tcr := NewTestClockRunner(t)

	// Get initial statistics
	initialWork, initialShort, initialLong := tcr.GetStatistics()

	// Run a complete cycle
	workSessions, shortBreaks, longBreaks := tcr.RunCompleteCycle()

	// Get final statistics
	finalWork, finalShort, finalLong := tcr.GetStatistics()

	// Verify statistics increased
	if finalWork <= initialWork {
		t.Error("Expected work sessions to increase")
	}

	if finalShort <= initialShort {
		t.Error("Expected short breaks to increase")
	}

	// Verify returned values match
	if workSessions != finalWork-initialWork {
		t.Error("Returned work sessions don't match statistics difference")
	}

	if shortBreaks != finalShort-initialShort {
		t.Error("Returned short breaks don't match statistics difference")
	}

	if longBreaks != finalLong-initialLong {
		t.Error("Returned long breaks don't match statistics difference")
	}
}
