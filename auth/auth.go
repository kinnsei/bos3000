package auth

import (
	"encore.dev/beta/auth"
	"encore.dev/storage/sqldb"
)

var db = sqldb.NewDatabase("auth", sqldb.DatabaseConfig{
	Migrations: "./migrations",
})

var secrets struct {
	JWTSecret string
}

//encore:service
type Service struct {
	jwtSecret []byte
}

func initService() (*Service, error) {
	return &Service{
		jwtSecret: []byte(secrets.JWTSecret),
	}, nil
}

// AuthData is returned by the auth handler and available to all endpoints.
type AuthData struct {
	UserID   int64  `json:"user_id"`
	Role     string `json:"role"`
	Username string `json:"username"`
}

// AuthParams defines the locations where auth credentials can be found.
type AuthParams struct {
	SessionCookie string `cookie:"session"`
	Authorization string `header:"Authorization"`
	APIKey        string `query:"api_key"`
}

// Data returns the AuthData for the current authenticated user.
func Data() *AuthData {
	data, _ := auth.Data().(*AuthData)
	return data
}
