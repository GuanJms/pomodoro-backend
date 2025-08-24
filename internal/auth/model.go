package auth

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	newUserRole = "USER"
	// JWT expiration time - 24 hours
	jwtExpirationHours = 24
	// JWT secret key (in production, this should be from environment variable) from .env file
)

var jwtSecretKey = os.Getenv("JWT_SECRET")

// Context keys for storing user information
type contextKey string

const (
	userIDKey   contextKey = "user_id"
	usernameKey contextKey = "username"
	roleKey     contextKey = "role"
)

type JWTClaims struct {
	Username string `json:"username"`
	UserID   string `json:"user_id"`
	jwt.RegisteredClaims
}

func NewAuthRepository(conn *pgxpool.Pool) AuthRepository {
	if conn == nil {
		return nil
	} else {
		return NewPostgresRepository(conn)
	}
}

type User struct {
	ID           *string    `json:"id,omitempty"`
	Username     *string    `json:"username,omitempty"`
	Password     *string    `json:"password,omitempty"`
	PasswordHash *string    `json:"password_hash,omitempty"`
	Email        *string    `json:"email,omitempty"`
	Role         *string    `json:"role,omitempty"`
	CreatedAt    *time.Time `json:"created_at,omitempty"`
}

type NewUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type UserLoginCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserLoginResponse struct {
	Token string `json:"token"`
}

type UserRegistrationResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func NewUserFromRequest(req *NewUserRequest) *User {
	role := newUserRole
	return &User{
		Username: &req.Username,
		Password: &req.Password,
		Role:     &role,
	}
}

// NewAdminUserFromRequest creates a new user with ADMIN role
func NewAdminUserFromRequest(req *NewUserRequest) *User {
	role := "ADMIN"
	return &User{
		Username: &req.Username,
		Password: &req.Password,
		Role:     &role,
	}
}

// GenerateJWT generates a JWT token for the given user
func GenerateJWT(user *User) (string, error) {
	if user.ID == nil || user.Username == nil {
		return "", jwt.ErrInvalidKey
	}

	expirationTime := time.Now().Add(jwtExpirationHours * time.Hour)

	claims := &JWTClaims{
		Username: *user.Username,
		UserID:   *user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "pomodoro-service",
			Subject:   *user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateJWT validates and parses a JWT token
func ValidateJWT(tokenString string) (*JWTClaims, error) {
	claims := &JWTClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(jwtSecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrInvalidKey
	}

	return claims, nil
}

// GetUserFromContext extracts user information from request context
func GetUserFromContext(ctx context.Context) (userID, username, role string, ok bool) {
	userIDVal := ctx.Value(userIDKey)
	usernameVal := ctx.Value(usernameKey)
	roleVal := ctx.Value(roleKey)

	if userIDVal == nil || usernameVal == nil || roleVal == nil {
		return "", "", "", false
	}

	userID, ok1 := userIDVal.(string)
	username, ok2 := usernameVal.(string)
	role, ok3 := roleVal.(string)

	if !ok1 || !ok2 || !ok3 {
		return "", "", "", false
	}

	return userID, username, role, true
}

// Role-based middleware functions

// RequireUserRole creates middleware that requires a specific user role
func RequireUserRole(repo AuthRepository, requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// First check if user is authenticated
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Extract token from "Bearer <token>" format
			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := tokenParts[1]
			claims, err := ValidateJWT(tokenString)
			if err != nil {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Get current user info from database to get the most up-to-date role
			user, err := repo.GetUserInfo(claims.Username)
			if err != nil {
				http.Error(w, "Failed to get user information", http.StatusInternalServerError)
				return
			}

			// Check if user has required role (using current role from database)
			if user.Role == nil || *user.Role != requiredRole {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			// Add user information to request context
			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			ctx = context.WithValue(ctx, usernameKey, claims.Username)
			ctx = context.WithValue(ctx, roleKey, *user.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdminRole creates middleware that requires ADMIN role
func RequireAdminRole(repo AuthRepository) func(http.Handler) http.Handler {
	return RequireUserRole(repo, "ADMIN")
}

// RequireAnyUserRole creates middleware that allows USER or ADMIN roles
func RequireAnyUserRole(repo AuthRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// First check if user is authenticated
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Extract token from "Bearer <token>" format
			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := tokenParts[1]
			claims, err := ValidateJWT(tokenString)
			if err != nil {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Get current user info from database to get the most up-to-date role
			user, err := repo.GetUserInfo(claims.Username)
			if err != nil {
				http.Error(w, "Failed to get user information", http.StatusInternalServerError)
				return
			}

			// Check if user has USER or ADMIN role (using current role from database)
			if user.Role == nil || (*user.Role != "USER" && *user.Role != "ADMIN") {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			// Add user information to request context
			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			ctx = context.WithValue(ctx, usernameKey, claims.Username)
			ctx = context.WithValue(ctx, roleKey, *user.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
