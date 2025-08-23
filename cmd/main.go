package main

import (
	"fmt"
	"log"
	"net/http"
	"pomodoroService/internal/clock"
	"time"
)

// var webPort = os.Getenv("WEB_PORT")
var webPort = "8080"
var redisAddr = "localhost:6379"

type Config struct {
	PomodoroSetting PomodoroSetting
	ClockRunner     *clock.ClockRunner
}

type PomodoroSetting struct {
	// read from default env
	WorkTimeDuration   int                `json:"workTimeDuration"`
	ShortBreakDuration int                `json:"shortBreakDuration"`
	LongBreakDuration  int                `json:"longBreakDuration"`
	Scheduling         []clock.ClockState `json:"scheduling"`
}

func main() {
	// Initialize clock runner with Redis persistence
	clockRunner, err := clock.NewClockRunnerWithRedis(redisAddr)
	if err != nil {
		log.Printf("Warning: failed to initialize Redis persistence, falling back to in-memory: %v", err)
		clockRunner = clock.NewClockRunner()
	}

	app := Config{
		PomodoroSetting: defaultPomodoroSetting(),
		ClockRunner:     clockRunner,
	}
	app.init()
	log.Printf("Starting pomodoro service on port %s\n", webPort)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.routes(),
	}

	err = srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

func (app *Config) init() {
	// utils.InitializeSecret()

	// Store the current session before setting durations/schedule
	// This preserves the session state loaded from Redis
	currentSession := app.ClockRunner.GetCurrentSession()
	log.Printf("🔄 Server startup: loaded currentSession=%d from Redis", currentSession)

	app.ClockRunner.SetDurations(
		time.Duration(app.PomodoroSetting.WorkTimeDuration)*time.Minute,
		time.Duration(app.PomodoroSetting.ShortBreakDuration)*time.Minute,
		time.Duration(app.PomodoroSetting.LongBreakDuration)*time.Minute,
	)

	err := app.ClockRunner.SetSchedule(app.PomodoroSetting.Scheduling)
	if err != nil {
		log.Fatalf("failed to set schedule: %v", err)
	}

	// After setting schedule, session gets reset to 0
	log.Printf("📝 After SetSchedule, currentSession reset to: %d", app.ClockRunner.GetCurrentSession())

	// Restore the session state that was loaded from Redis
	// This prevents the schedule reset from overriding the persisted state
	app.ClockRunner.GetSessionManager().SetCurrentSession(currentSession)
	log.Printf("✅ Restored session state from Redis: currentSession=%d", currentSession)
}

func defaultPomodoroSetting() PomodoroSetting {
	// workTimeDuration, _ := strconv.Atoi(os.Getenv("WORK_TIME_DURATION"))
	// shortBreakDuration, _ := strconv.Atoi(os.Getenv("SHORT_BREAK_DURATION"))
	// longBreakDuration, _ := strconv.Atoi(os.Getenv("LONG_BREAK_DURATION"))
	// scheduling := os.Getenv("SCHEDULING")

	// workTimeDuration := 25
	// shortBreakDuration := 5
	// longBreakDuration := 15
	workTimeDuration := 1
	shortBreakDuration := 1
	longBreakDuration := 1
	schedulingString := `W-SB-W-SB-W-LB`
	scheduling, err := clock.ParseScheduling(schedulingString)
	if err != nil {
		log.Fatalf("failed to parse scheduling: %v", err)
	}

	return PomodoroSetting{
		WorkTimeDuration:   workTimeDuration,
		ShortBreakDuration: shortBreakDuration,
		LongBreakDuration:  longBreakDuration,
		Scheduling:         scheduling,
	}
}
