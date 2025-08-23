package test

import (
	"sync"
	"testing"
	"time"

	. "pomodoroService/internal/clock"
)

func TestModularComponents(t *testing.T) {
	// Test StateManager
	t.Run("StateManager", func(t *testing.T) {
		sm := NewStateManager()

		if sm.GetState() != StateIdle {
			t.Errorf("Expected initial state to be StateIdle, got %s", sm.GetState())
		}

		if !sm.IsIdle() {
			t.Error("Expected state manager to be idle initially")
		}

		if !sm.CanStart() {
			t.Error("Expected state manager to allow starting from idle")
		}

		sm.SetState(StateWorking)
		if sm.GetState() != StateWorking {
			t.Errorf("Expected state to be StateWorking, got %s", sm.GetState())
		}

		if !sm.IsRunning() {
			t.Error("Expected state manager to be running")
		}

		if !sm.CanPause() {
			t.Error("Expected state manager to allow pausing when running")
		}
	})

	// Test SessionManager
	t.Run("SessionManager", func(t *testing.T) {
		sm := NewSessionManager()

		if sm.GetCurrentSession() != 0 {
			t.Errorf("Expected initial session to be 0, got %d", sm.GetCurrentSession())
		}

		if sm.GetTotalSessions() != 8 {
			t.Errorf("Expected total sessions to be 8, got %d", sm.GetTotalSessions())
		}

		state := sm.GetCurrentSessionState()
		if state != StateWorking {
			t.Errorf("Expected first session to be StateWorking, got %s", state)
		}

		duration := sm.GetCurrentSessionDuration()
		if duration != 25*time.Minute {
			t.Errorf("Expected work duration to be 25 minutes, got %v", duration)
		}

		// Test custom durations
		sm.SetDurations(10*time.Minute, 2*time.Minute, 5*time.Minute)
		duration = sm.GetCurrentSessionDuration()
		if duration != 10*time.Minute {
			t.Errorf("Expected custom work duration to be 10 minutes, got %v", duration)
		}
	})

	// Test TimerManager
	t.Run("TimerManager", func(t *testing.T) {
		tm := NewTimerManager()

		if tm.IsRunning() {
			t.Error("Expected timer to not be running initially")
		}

		remaining := tm.GetTimeRemaining()
		if remaining != 0 {
			t.Errorf("Expected remaining time to be 0, got %v", remaining)
		}

		// Test timer start
		var tickCount int
		var completedState ClockState

		tm.StartTimer(100*time.Millisecond, StateWorking,
			func(remaining time.Duration) {
				tickCount++
			},
			func(state ClockState) {
				completedState = state
			})

		if !tm.IsRunning() {
			t.Error("Expected timer to be running after start")
		}

		// Wait for completion
		time.Sleep(150 * time.Millisecond)

		if completedState != StateWorking {
			t.Errorf("Expected completed state to be StateWorking, got %s", completedState)
		}

		if tickCount == 0 {
			t.Error("Expected at least one tick")
		}
	})

	// Test StatisticsManager
	t.Run("StatisticsManager", func(t *testing.T) {
		sm := NewStatisticsManager()

		work, short, long := sm.GetStatistics()
		if work != 0 || short != 0 || long != 0 {
			t.Errorf("Expected initial statistics to be 0, got %d, %d, %d", work, short, long)
		}

		// Record some sessions
		sm.RecordSession(StateWorking, 25*time.Minute)
		sm.RecordSession(StateShortBreak, 5*time.Minute)
		sm.RecordSession(StateWorking, 25*time.Minute)

		work, short, long = sm.GetStatistics()
		if work != 2 {
			t.Errorf("Expected 2 work sessions, got %d", work)
		}
		if short != 1 {
			t.Errorf("Expected 1 short break, got %d", short)
		}
		if long != 0 {
			t.Errorf("Expected 0 long breaks, got %d", long)
		}

		history := sm.GetSessionHistory()
		if len(history) != 3 {
			t.Errorf("Expected 3 session records, got %d", len(history))
		}
	})

	// Test TimeFormatter
	t.Run("TimeFormatter", func(t *testing.T) {
		tf := NewTimeFormatter()

		formatted := tf.FormatDuration(65 * time.Second)
		if formatted != "01:05" {
			t.Errorf("Expected '01:05', got %s", formatted)
		}

		formatted = tf.FormatDuration(0)
		if formatted != "00:00" {
			t.Errorf("Expected '00:00', got %s", formatted)
		}

		longFormatted := tf.FormatDurationLong(90 * time.Minute)
		if longFormatted != "1 hours 30 minutes" {
			t.Errorf("Expected '1 hours 30 minutes', got %s", longFormatted)
		}
	})

	// Test ClockUtils
	t.Run("ClockUtils", func(t *testing.T) {
		cu := NewClockUtils()

		if !cu.IsValidDuration(25 * time.Minute) {
			t.Error("Expected 25 minutes to be valid")
		}

		if cu.IsValidDuration(5 * time.Hour) {
			t.Error("Expected 5 hours to be invalid")
		}

		progress := cu.CalculateSessionProgress(15*time.Minute, 25*time.Minute)
		if progress != 60.0 {
			t.Errorf("Expected 60%% progress, got %.1f", progress)
		}

		schedule := []ClockState{StateWorking, StateShortBreak, StateWorking}
		err := cu.ValidateSchedule(schedule)
		if err != nil {
			t.Errorf("Expected valid schedule, got error: %v", err)
		}

		summary := cu.GetScheduleSummary(schedule)
		if summary[StateWorking] != 2 {
			t.Errorf("Expected 2 work sessions in summary, got %d", summary[StateWorking])
		}
	})
}

func TestModularClockRunner(t *testing.T) {
	cr := NewClockRunner()

	// Test initial state
	if cr.GetState() != StateIdle {
		t.Errorf("Expected initial state to be StateIdle, got %s", cr.GetState())
	}

	if !cr.IsIdle() {
		t.Error("Expected clock to be idle initially")
	}

	// Test setting durations
	cr.SetDurations(1*time.Second, 500*time.Millisecond, 750*time.Millisecond)

	// Test callbacks
	var stateChanges []ClockState
	var ticks []time.Duration
	var completions []ClockState

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

	// Test starting
	err := cr.Start()
	if err != nil {
		t.Errorf("Expected no error when starting, got %v", err)
	}

	if cr.GetState() != StateWorking {
		t.Errorf("Expected state to be StateWorking after start, got %s", cr.GetState())
	}

	if !cr.IsRunning() {
		t.Error("Expected clock to be running after start")
	}

	// Wait a bit for ticks
	time.Sleep(200 * time.Millisecond)

	if len(ticks) == 0 {
		t.Error("Expected at least one tick")
	}

	// Test pausing
	err = cr.Pause()
	if err != nil {
		t.Errorf("Expected no error when pausing, got %v", err)
	}

	if cr.GetState() != StatePaused {
		t.Errorf("Expected state to be StatePaused after pause, got %s", cr.GetState())
	}

	if !cr.IsPaused() {
		t.Error("Expected clock to be paused")
	}

	// Test resuming
	err = cr.Start()
	if err != nil {
		t.Errorf("Expected no error when resuming, got %v", err)
	}

	if cr.GetState() != StateWorking {
		t.Errorf("Expected state to be StateWorking after resume, got %s", cr.GetState())
	}

	// Test stopping
	err = cr.Stop()
	if err != nil {
		t.Errorf("Expected no error when stopping, got %v", err)
	}

	if cr.GetState() != StateIdle {
		t.Errorf("Expected state to be StateIdle after stop, got %s", cr.GetState())
	}

	if !cr.IsIdle() {
		t.Error("Expected clock to be idle after stop")
	}

	// Test state changes were recorded
	if len(stateChanges) == 0 {
		t.Error("Expected state changes to be recorded")
	}
}

func TestModularClockRunnerConcurrency(t *testing.T) {
	cr := NewClockRunner()
	cr.SetDurations(100*time.Millisecond, 50*time.Millisecond, 75*time.Millisecond)

	var wg sync.WaitGroup
	concurrent := 10

	// Test concurrent access
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Read operations
			_ = cr.GetState()
			_ = cr.GetCurrentSession()
			_ = cr.GetTimeRemaining()
			_ = cr.IsRunning()
			_ = cr.IsPaused()
			_ = cr.IsIdle()
		}()
	}

	wg.Wait()

	// Test concurrent write operations (should be safe)
	cr.Start()

	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _ = cr.GetStatistics()
			_ = cr.GetSessionHistory()
		}()
	}

	wg.Wait()

	cr.Stop()
}
