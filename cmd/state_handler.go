package main

import (
	"log"
	"net/http"
)

type SystemHandler struct {
}

func (h *SystemHandler) GetSystemState(w http.ResponseWriter, r *http.Request) {
	log.Panic("Not implemented")
}

func (h *SystemHandler) StartNewPomodoro(w http.ResponseWriter, r *http.Request) {
	log.Panic("Not implemented")
}
