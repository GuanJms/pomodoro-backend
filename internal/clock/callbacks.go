package clock

import (
	"log"
	"time"
)

func onComplete(cr *ClockRunner, completedState ClockState, duration time.Duration) {
	log.Printf("▶️ onComplete - 1")
	// Record the completed session
	cr.statsManager.RecordSession(completedState, duration)

	if cr.onComplete != nil {
		cr.onComplete(completedState)
	}

	log.Printf("Completed %s session %d/%d - 2",
		completedState, cr.sessionManager.GetCurrentSession(), cr.sessionManager.GetTotalSessions())

	// Move to next session
	log.Printf("Moving to next session from %d - 3", cr.sessionManager.GetCurrentSession())
	hasNextSession := cr.sessionManager.NextSession()
	log.Printf("Next session available: %v, current session now: %d - 4", hasNextSession, cr.sessionManager.GetCurrentSession())
	if !hasNextSession {
		// Completed all sessions - set to idle and save state
		cr.stateManager.SetState(StateIdle)
		if cr.onStateChange != nil {
			cr.onStateChange(StateIdle)
		}
		// Save idle state to Redis immediately
		cr.saveStateToRedis()
		log.Println("Completed all pomodoro sessions!")
		return
	}

	// Start next session - this will save state to Redis
	cr.startNewSession()

	// Ensure the new session state is saved to Redis immediately
	cr.saveStateToRedis()
}

func onTick(cr *ClockRunner, remaining time.Duration) {
	if cr.onTick != nil {
		cr.onTick(remaining)
	}
}
