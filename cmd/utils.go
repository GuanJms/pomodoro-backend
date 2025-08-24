package main

import (
	"context"
	"log"
	"pomodoroService/internal/clock"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PomodoroSetting struct {
	// read from default env
	WorkTimeDuration   int                `json:"workTimeDuration"`
	ShortBreakDuration int                `json:"shortBreakDuration"`
	LongBreakDuration  int                `json:"longBreakDuration"`
	Scheduling         []clock.ClockState `json:"scheduling"`
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

var counts = 0

func connectToDB() *pgxpool.Pool {

	for {
		// keep connecting to the database
		connection, err := openDB(dsn)
		if err != nil {
			log.Printf("Postgres is not yet ready")
			counts++
		} else {
			log.Printf("Connected to Postgres!")
			return connection
		}

		if counts > 10 {
			log.Println(err)
			return nil
		}

		log.Println("Backing off for two seconds...")
		time.Sleep(2 * time.Second)
	}
}

func openDB(dsn string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	err = pool.Ping(context.Background())
	if err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
