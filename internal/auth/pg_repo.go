package auth

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	authdb "pomodoroService/internal/auth/gen"
)

const (
	dbTimeout = time.Second * 3
)

var (
	// Email validation regex - standard email format
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

var ErrNoContent error

type PostgresRepository struct {
	Conn    *pgxpool.Pool
	Queries *authdb.Queries
}

func NewPostgresRepository(conn *pgxpool.Pool) AuthRepository {
	ErrNoContent = errors.New("no content error")
	return &PostgresRepository{
		Conn:    conn,
		Queries: authdb.New(conn),
	}
}

// Helper functions to convert between sqlc generated models and our User model

func stringPtr(s string) *string {
	return &s
}

func convertUserRoleToString(role authdb.UserRole) string {
	return string(role)
}

func convertStringToUserRole(role string) authdb.UserRole {
	switch role {
	case "ADMIN":
		return authdb.UserRoleADMIN
	case "USER":
		return authdb.UserRoleUSER
	default:
		return authdb.UserRoleUSER
	}
}

func convertUserToCreateUserParams(user *User) (authdb.CreateUserParams, error) {
	if user.Username == nil || user.Email == nil || user.PasswordHash == nil || user.Role == nil {
		return authdb.CreateUserParams{}, fmt.Errorf("missing required fields")
	}

	// Validate email format before converting
	if !emailRegex.MatchString(*user.Email) {
		return authdb.CreateUserParams{}, fmt.Errorf("invalid email format")
	}

	return authdb.CreateUserParams{
		Username:     *user.Username,
		Email:        *user.Email,
		PasswordHash: *user.PasswordHash,
		Role:         convertStringToUserRole(*user.Role),
	}, nil
}

func convertCreateUserRowToUser(row authdb.CreateUserRow) *User {
	id := row.ID.String()
	username := row.Username
	email := row.Email
	role := convertUserRoleToString(row.Role)
	createdAt := row.CreatedAt.Time.UTC()

	return &User{
		ID:        &id,
		Username:  &username,
		Email:     &email,
		Role:      &role,
		CreatedAt: &createdAt,
	}
}

func convertGetUserByUsernameRowToUser(row authdb.GetUserByUsernameRow) *User {
	id := row.ID.String()
	username := row.Username
	email := row.Email
	role := convertUserRoleToString(row.Role)
	createdAt := row.CreatedAt.Time.UTC()

	return &User{
		ID:        &id,
		Username:  &username,
		Email:     &email,
		Role:      &role,
		CreatedAt: &createdAt,
	}
}

// ValidateNewUser validates a new user before creation
func (p *PostgresRepository) ValidateNewUser(user *User) error {
	if user.Username == nil || strings.TrimSpace(*user.Username) == "" {
		return fmt.Errorf("username is required")
	}

	if user.Email == nil || strings.TrimSpace(*user.Email) == "" {
		return fmt.Errorf("email is required")
	}

	// Validate email format
	if !emailRegex.MatchString(*user.Email) {
		return fmt.Errorf("invalid email format")
	}

	// Check if username already exists
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	_, err := p.Queries.GetUserByUsername(ctx, *user.Username)
	if err == nil {
		return fmt.Errorf("username already exists")
	}

	// Check if email already exists
	_, err = p.Queries.GetUserByEmail(ctx, *user.Email)
	if err == nil {
		return fmt.Errorf("email already exists")
	}

	// If the error is not "no rows found", it's a different database error
	if !strings.Contains(err.Error(), "no rows in result set") {
		return fmt.Errorf("database error during validation: %w", err)
	}

	return nil
}

func (p *PostgresRepository) CreateUser(user *User) error {
	if user.Password == nil {
		return fmt.Errorf("password is required")
	}

	// Validate user data before proceeding
	if err := p.ValidateNewUser(user); err != nil {
		return err
	}

	// Hash the password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(*user.Password), 12)
	if err != nil {
		return err
	}

	// Set the password hash
	user.PasswordHash = stringPtr(string(passwordHash))

	// Convert user to sqlc params
	params, err := convertUserToCreateUserParams(user)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// Use sqlc generated function
	result, err := p.Queries.CreateUser(ctx, params)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok {
			if pgErr.Code == "23505" {
				return fmt.Errorf("username or email already exists")
			}
		}
		return err
	}

	// Convert result back to our User model
	convertedUser := convertCreateUserRowToUser(result)
	user.ID = convertedUser.ID
	user.CreatedAt = convertedUser.CreatedAt

	return nil
}

func (p *PostgresRepository) AuthenticateUser(cred *UserLoginCredentials) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// Get password hash using sqlc generated function
	passwordHash, err := p.Queries.GetPasswordHashByUsername(ctx, cred.Username)
	if err != nil {
		return false, errors.New("invalid credentials")
	}

	// Compare password hash
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(cred.Password)); err != nil {
		return false, errors.New("invalid credentials")
	}

	// Successful authentication
	return true, nil
}

func (p *PostgresRepository) GetUserInfo(username string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// Use sqlc generated function
	result, err := p.Queries.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	// Convert result to our User model
	user := convertGetUserByUsernameRowToUser(result)
	return user, nil
}
