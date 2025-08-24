package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"pomodoroService/internal/auth"
	"pomodoroService/internal/clock"

	"github.com/jackc/pgx/v5/pgxpool"
)

// var webPort = "8080"             // os.Getenv("WEB_PORT") or default
// var redisAddr = "localhost:6379" // os.Getenv("REDIS_ADDR") or default
// var dsn = "postgres://user:password@localhost:5435/pomodoro_db?sslmode=disable"

var webPort = os.Getenv("WEB_PORT")
var redisAddr = os.Getenv("REDIS_ADDR")
var dsn = os.Getenv("DSN")

type Config struct {
	PomodoroSetting PomodoroSetting
	ClockRunner     *clock.ClockRunner
	AuthRepo        auth.AuthRepository
}

func main() {
	// Initialize clock runner with Redis persistence
	clockRunner, err := clock.NewClockRunnerWithRedis(redisAddr)
	if err != nil {
		log.Printf("Warning: failed to initialize Redis persistence, falling back to in-memory: %v", err)
		clockRunner = clock.NewClockRunner()
	}
	conn := connectToDB()
	if conn == nil {
		log.Panic("Can't connect to Postgres")
	}

	app := Config{
		PomodoroSetting: defaultPomodoroSetting(),
		ClockRunner:     clockRunner,
	}
	app.setupRepo(conn)
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

func (app *Config) setupRepo(conn *pgxpool.Pool) {
	app.AuthRepo = auth.NewAuthRepository(conn)
}
