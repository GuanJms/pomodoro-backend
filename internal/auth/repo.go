package auth

type AuthRepository interface {
	CreateUser(user *User) error
	AuthenticateUser(credentials *UserLoginCredentials) (bool, error)
	GetUserInfo(username string) (*User, error)
}
