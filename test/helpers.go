package test

import (
	"pomodoroService/internal/clock"
	"testing"
	"time"
)

// TestClockRunner is a helper struct for testing clock runner functionality
type TestClockRunner struct {
	*clock.ClockRunner
	t *testing.T
}

// NewTestClockRunner creates a new test clock runner with longer durations for reliability
func NewTestClockRunner(t *testing.T) *TestClockRunner {
	cr := clock.NewClockRunner()
	cr.SetDurations(200*time.Millisecond, 100*time.Millisecond, 150*time.Millisecond)

	return &TestClockRunner{
		ClockRunner: cr,
		t:           t,
	}
}

// AssertState asserts that the clock runner is in the expected state
func (tcr *TestClockRunner) AssertState(expected clock.ClockState) {
	actual := tcr.GetState()
	if actual != expected {
		tcr.t.Errorf("Expected state to be %s, got %s", expected, actual)
	}
}

// AssertTimeRemaining asserts that the remaining time is within expected bounds
func (tcr *TestClockRunner) AssertTimeRemaining(min, max time.Duration) {
	remaining := tcr.GetTimeRemaining()
	if remaining < min || remaining > max {
		tcr.t.Errorf("Expected remaining time to be between %v and %v, got %v", min, max, remaining)
	}
}

// AssertSession asserts that the current session is as expected
func (tcr *TestClockRunner) AssertSession(expected int) {
	actual := tcr.GetCurrentSession()
	if actual != expected {
		tcr.t.Errorf("Expected session to be %d, got %d", expected, actual)
	}
}

// WaitForState waits for the clock runner to reach a specific state with timeout
func (tcr *TestClockRunner) WaitForState(expected clock.ClockState, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if tcr.GetState() == expected {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	tcr.t.Errorf("Timeout waiting for state %s", expected)
	return false
}

// WaitForSession waits for the clock runner to reach a specific session with timeout
func (tcr *TestClockRunner) WaitForSession(expected int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if tcr.GetCurrentSession() == expected {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	tcr.t.Errorf("Timeout waiting for session %d", expected)
	return false
}

// CompleteSession waits for the current session to complete
func (tcr *TestClockRunner) CompleteSession() {
	initialSession := tcr.GetCurrentSession()
	initialState := tcr.GetState()

	// Wait for session to change
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if tcr.GetCurrentSession() != initialSession || tcr.GetState() != initialState {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	tcr.t.Errorf("Timeout waiting for session to complete")
}

// RunCompleteCycle runs a complete pomodoro cycle and returns statistics
func (tcr *TestClockRunner) RunCompleteCycle() (int, int, int) {
	initialWork, initialShort, initialLong := tcr.GetStatistics()

	// Start the clock
	err := tcr.Start()
	if err != nil {
		tcr.t.Fatalf("Failed to start clock: %v", err)
	}

	// Wait for complete cycle (4*200ms + 3*100ms + 1*150ms = 1.25s + buffer)
	time.Sleep(2 * time.Second)

	// Get final statistics
	finalWork, finalShort, finalLong := tcr.GetStatistics()

	return finalWork - initialWork, finalShort - initialShort, finalLong - initialLong
}

// StateTracker tracks state changes during testing
type StateTracker struct {
	states      []clock.ClockState
	completions []clock.ClockState
	ticks       []time.Duration
}

// NewStateTracker creates a new state tracker
func NewStateTracker() *StateTracker {
	return &StateTracker{
		states:      make([]clock.ClockState, 0),
		completions: make([]clock.ClockState, 0),
		ticks:       make([]time.Duration, 0),
	}
}

// OnStateChange callback for state changes
func (st *StateTracker) OnStateChange(state clock.ClockState) {
	st.states = append(st.states, state)
}

// OnTick callback for ticks
func (st *StateTracker) OnTick(remaining time.Duration) {
	st.ticks = append(st.ticks, remaining)
}

// OnComplete callback for session completions
func (st *StateTracker) OnComplete(completedState clock.ClockState) {
	st.completions = append(st.completions, completedState)
}

// GetStates returns all recorded states
func (st *StateTracker) GetStates() []clock.ClockState {
	return st.states
}

// GetCompletions returns all recorded completions
func (st *StateTracker) GetCompletions() []clock.ClockState {
	return st.completions
}

// GetTicks returns all recorded ticks
func (st *StateTracker) GetTicks() []time.Duration {
	return st.ticks
}

// CountState counts occurrences of a specific state
func (st *StateTracker) CountState(state clock.ClockState) int {
	count := 0
	for _, s := range st.states {
		if s == state {
			count++
		}
	}
	return count
}

// CountCompletion counts occurrences of a specific completion
func (st *StateTracker) CountCompletion(state clock.ClockState) int {
	count := 0
	for _, s := range st.completions {
		if s == state {
			count++
		}
	}
	return count
}

// AssertStateSequence asserts that the states occurred in the expected sequence
func (st *StateTracker) AssertStateSequence(t *testing.T, expected []clock.ClockState) {
	if len(st.states) < len(expected) {
		t.Errorf("Expected at least %d states, got %d", len(expected), len(st.states))
		return
	}

	for i, expectedState := range expected {
		if i >= len(st.states) {
			t.Errorf("Expected state %s at position %d, but no more states recorded", expectedState, i)
			return
		}
		if st.states[i] != expectedState {
			t.Errorf("Expected state %s at position %d, got %s", expectedState, i, st.states[i])
		}
	}
}

// TestScenario represents a test scenario with expected outcomes
type TestScenario struct {
	Name       string
	Setup      func(*TestClockRunner)
	Actions    func(*TestClockRunner)
	Assertions func(*TestClockRunner, *StateTracker)
	Duration   time.Duration
}

// RunScenario runs a test scenario
func RunScenario(t *testing.T, scenario TestScenario) {
	t.Run(scenario.Name, func(t *testing.T) {
		tcr := NewTestClockRunner(t)
		tracker := NewStateTracker()

		// Set up callbacks
		tcr.SetCallbacks(
			tracker.OnStateChange,
			tracker.OnTick,
			tracker.OnComplete,
		)

		// Run setup
		if scenario.Setup != nil {
			scenario.Setup(tcr)
		}

		// Run actions
		if scenario.Actions != nil {
			scenario.Actions(tcr)
		}

		// Wait for scenario duration
		if scenario.Duration > 0 {
			time.Sleep(scenario.Duration)
		}

		// Run assertions
		if scenario.Assertions != nil {
			scenario.Assertions(tcr, tracker)
		}
	})
}

// Common test scenarios
var (
	// BasicWorkflowScenario tests the basic pomodoro workflow
	BasicWorkflowScenario = TestScenario{
		Name: "Basic Workflow",
		Actions: func(tcr *TestClockRunner) {
			tcr.Start()
		},
		Duration: 2 * time.Second, // Wait for complete cycle (4*200ms + 3*100ms + 1*150ms = 1.25s)
		Assertions: func(tcr *TestClockRunner, tracker *StateTracker) {
			if len(tracker.GetStates()) == 0 {
				tcr.t.Error("Expected state changes to be recorded")
			}
			if len(tracker.GetCompletions()) == 0 {
				tcr.t.Error("Expected completions to be recorded")
			}
			tcr.AssertState(clock.StateIdle)
		},
	}

	// PauseResumeScenario tests pausing and resuming
	PauseResumeScenario = TestScenario{
		Name: "Pause Resume",
		Actions: func(tcr *TestClockRunner) {
			tcr.Start()
			time.Sleep(50 * time.Millisecond)
			tcr.Pause()
			time.Sleep(50 * time.Millisecond)
			tcr.Start()
		},
		Duration: 300 * time.Millisecond,
		Assertions: func(tcr *TestClockRunner, tracker *StateTracker) {
			if tracker.CountState(clock.StatePaused) == 0 {
				tcr.t.Error("Expected pause state to be recorded")
			}
		},
	}

	// SkipScenario tests skipping sessions
	SkipScenario = TestScenario{
		Name: "Skip Sessions",
		Actions: func(tcr *TestClockRunner) {
			tcr.Start()
			time.Sleep(10 * time.Millisecond)
			tcr.Skip()
			time.Sleep(10 * time.Millisecond)
			tcr.Skip()
		},
		Duration: 200 * time.Millisecond,
		Assertions: func(tcr *TestClockRunner, tracker *StateTracker) {
			if len(tracker.GetCompletions()) < 2 {
				tcr.t.Error("Expected at least 2 skipped sessions")
			}
		},
	}
)
