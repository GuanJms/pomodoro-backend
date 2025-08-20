package main

import (
	"fmt"
	"log"
	"net/http"
)

// var webPort = os.Getenv("WEB_PORT")
var webPort = "8080"

type Config struct {
	PomodoroSetting PomodoroSetting
}

type PomodoroSetting struct {
	// read from default env
	WorkTimeDuration   int    `json:"workTimeDuration"`
	ShortBreakDuration int    `json:"shortBreakDuration"`
	LongBreakDuration  int    `json:"longBreakDuration"`
	Scheduling         string `json:"scheduling"`
}

func main() {
	app := Config{
		PomodoroSetting: defaultPomodoroSetting(),
	}
	// app.init()
	log.Printf("Starting pomodoro service on port %s\n", webPort)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.routes(),
	}

	err := srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

func (*Config) init() {
	// utils.InitializeSecret()
}

func defaultPomodoroSetting() PomodoroSetting {
	// workTimeDuration, _ := strconv.Atoi(os.Getenv("WORK_TIME_DURATION"))
	// shortBreakDuration, _ := strconv.Atoi(os.Getenv("SHORT_BREAK_DURATION"))
	// longBreakDuration, _ := strconv.Atoi(os.Getenv("LONG_BREAK_DURATION"))
	// scheduling := os.Getenv("SCHEDULING")

	workTimeDuration := 25
	shortBreakDuration := 5
	longBreakDuration := 15
	scheduling := "W-SB-W-SB-W-SB-W-LB"

	return PomodoroSetting{
		WorkTimeDuration:   workTimeDuration,
		ShortBreakDuration: shortBreakDuration,
		LongBreakDuration:  longBreakDuration,
		Scheduling:         scheduling,
	}
}
