package main

import (
	"net/http"

	"pomodoroService/internal/auth"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func (app *Config) routes() http.Handler {
	mux := chi.NewRouter()

	mux.Use(cors.Handler(cors.Options{
		// AllowedOrigins: []string{"https://*", "http://*"},
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	mux.Use(middleware.Heartbeat("/ping"))

	clockHandler := NewClockHandler(app.ClockRunner)
	authHandler := NewAuthHandler(app.AuthRepo)

	// Clock routes with role-based access control
	// Basic users (USER role) can view system state
	mux.With(auth.RequireAnyUserRole(app.AuthRepo)).Get("/system/state", clockHandler.GetSystemState)
	// Only admins can start/modify the pomodoro system
	mux.With(auth.RequireAdminRole(app.AuthRepo)).Post("/system/start", clockHandler.StartNewPomodoro)

	// Authentication routes
	mux.Post("/auth/register", authHandler.RegisterUser)
	mux.Post("/auth/login", authHandler.LoginUser)

	// Admin registration (development/testing only)
	mux.Post("/auth/register-admin", authHandler.RegisterAdminUser)

	// Protected routes (require JWT token)
	mux.With(auth.RequireAnyUserRole(app.AuthRepo)).Get("/auth/profile", authHandler.GetProfile)

	return mux
}
