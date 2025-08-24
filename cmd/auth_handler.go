// Package main provides authentication handlers with role-based access control.
// For detailed documentation, see README.md
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"pomodoroService/internal/auth"
	"strings"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authRepo auth.AuthRepository
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authRepo auth.AuthRepository) *AuthHandler {
	return &AuthHandler{
		authRepo: authRepo,
	}
}

// RegisterUser handles user registration
func (h *AuthHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "Use POST method")
		return
	}

	var req auth.NewUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON", "Failed to parse request body")
		return
	}

	// Validate request
	if strings.TrimSpace(req.Username) == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Validation error", "Username is required")
		return
	}
	if strings.TrimSpace(req.Password) == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Validation error", "Password is required")
		return
	}
	if strings.TrimSpace(req.Email) == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Validation error", "Email is required")
		return
	}

	// Create user from request
	user := auth.NewUserFromRequest(&req)
	user.Email = &req.Email // Ensure email is set

	// Create user in repository
	if err := h.authRepo.CreateUser(user); err != nil {
		log.Printf("Failed to create user: %v", err)

		// Handle specific error messages
		if strings.Contains(err.Error(), "username already exists") ||
			strings.Contains(err.Error(), "email already exists") ||
			strings.Contains(err.Error(), "already exists") {
			h.writeErrorResponse(w, http.StatusConflict, "User already exists", err.Error())
			return
		}
		if strings.Contains(err.Error(), "invalid email format") {
			h.writeErrorResponse(w, http.StatusBadRequest, "Invalid email format", err.Error())
			return
		}
		if strings.Contains(err.Error(), "required") {
			h.writeErrorResponse(w, http.StatusBadRequest, "Validation error", err.Error())
			return
		}

		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create user", "Internal server error")
		return
	}

	// Generate JWT token for the newly created user
	token, err := auth.GenerateJWT(user)
	if err != nil {
		log.Printf("Failed to generate JWT token: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to generate token", "Internal server error")
		return
	}

	// Prepare response
	response := auth.UserRegistrationResponse{
		ID:       *user.ID,
		Username: *user.Username,
		Email:    *user.Email,
		Role:     *user.Role,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	// Return both user info and token
	fullResponse := map[string]interface{}{
		"user":  response,
		"token": token,
	}

	if err := json.NewEncoder(w).Encode(fullResponse); err != nil {
		log.Printf("Failed to encode response: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to encode response", "Internal server error")
		return
	}
}

// LoginUser handles user authentication
func (h *AuthHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	// log.Printf("LoginUser")
	if r.Method != http.MethodPost {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "Use POST method")
		return
	}

	var creds auth.UserLoginCredentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON", "Failed to parse request body")
		return
	}

	// Validate credentials
	if strings.TrimSpace(creds.Username) == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Validation error", "Username is required")
		return
	}
	if strings.TrimSpace(creds.Password) == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Validation error", "Password is required")
		return
	}

	// Authenticate user
	isAuthenticated, err := h.authRepo.AuthenticateUser(&creds)
	log.Printf("isAuthenticated: %v", isAuthenticated)
	if err != nil {
		log.Printf("Authentication error: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Authentication failed", "Internal server error")
		return
	}

	if !isAuthenticated {
		h.writeErrorResponse(w, http.StatusUnauthorized, "Invalid credentials", "Username or password is incorrect")
		return
	}

	// Get user info for JWT generation
	user, err := h.authRepo.GetUserInfo(creds.Username)
	// log.Printf("user: %v", user)
	if err != nil {
		log.Printf("Failed to get user info: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get user info", "Internal server error")
		return
	}

	// Generate JWT token
	token, err := auth.GenerateJWT(user)
	// log.Printf("token: %v", token)
	if err != nil {
		log.Printf("Failed to generate JWT token: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to generate token", "Internal server error")
		return
	}

	// Prepare response
	response := auth.UserLoginResponse{
		Token: token,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to encode response", "Internal server error")
		return
	}
}

// writeErrorResponse writes a standardized error response
func (h *AuthHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, errorMsg, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := auth.ErrorResponse{
		Error:   errorMsg,
		Message: message,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode error response: %v", err)
	}
}

// GetProfile is an example of a protected endpoint that requires JWT authentication
func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Get user information from context (set by JWT middleware)
	_, username, _, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "Unauthorized", "User information not found in context")
		return
	}

	// Get full user info from repository
	user, err := h.authRepo.GetUserInfo(username)
	if err != nil {
		log.Printf("Failed to get user info: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get user info", "Internal server error")
		return
	}

	// Return user profile
	profile := map[string]interface{}{
		"id":         *user.ID,
		"username":   *user.Username,
		"email":      *user.Email,
		"role":       *user.Role,
		"created_at": *user.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(profile); err != nil {
		log.Printf("Failed to encode profile response: %v", err)
	}
}

// RegisterAdminUser handles admin user registration
// WARNING: Development/testing only - disable in production
func (h *AuthHandler) RegisterAdminUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "Use POST method")
		return
	}

	var req auth.NewUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON", "Failed to parse request body")
		return
	}

	// Validate request
	if strings.TrimSpace(req.Username) == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Validation error", "Username is required")
		return
	}
	if strings.TrimSpace(req.Password) == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Validation error", "Password is required")
		return
	}
	if strings.TrimSpace(req.Email) == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Validation error", "Email is required")
		return
	}

	// Create admin user from request
	user := auth.NewAdminUserFromRequest(&req)
	user.Email = &req.Email // Ensure email is set

	// Create user in repository
	if err := h.authRepo.CreateUser(user); err != nil {
		log.Printf("Failed to create admin user: %v", err)

		// Handle specific error messages
		if strings.Contains(err.Error(), "username already exists") ||
			strings.Contains(err.Error(), "email already exists") ||
			strings.Contains(err.Error(), "already exists") {
			h.writeErrorResponse(w, http.StatusConflict, "User already exists", err.Error())
			return
		}
		if strings.Contains(err.Error(), "invalid email format") {
			h.writeErrorResponse(w, http.StatusBadRequest, "Invalid email format", err.Error())
			return
		}
		if strings.Contains(err.Error(), "required") {
			h.writeErrorResponse(w, http.StatusBadRequest, "Validation error", err.Error())
			return
		}

		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create admin user", "Internal server error")
		return
	}

	// Generate JWT token for the newly created admin user
	token, err := auth.GenerateJWT(user)
	if err != nil {
		log.Printf("Failed to generate JWT token: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to generate token", "Internal server error")
		return
	}

	// Prepare response
	response := auth.UserRegistrationResponse{
		ID:       *user.ID,
		Username: *user.Username,
		Email:    *user.Email,
		Role:     *user.Role,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	// Return both user info and token
	fullResponse := map[string]interface{}{
		"user":  response,
		"token": token,
	}

	if err := json.NewEncoder(w).Encode(fullResponse); err != nil {
		log.Printf("Failed to encode response: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to encode response", "Internal server error")
		return
	}
}
