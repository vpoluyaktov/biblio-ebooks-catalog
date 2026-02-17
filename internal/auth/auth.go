package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"biblio-ebooks-catalog/internal/db"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrSessionExpired     = errors.New("session expired")
	ErrUnauthorized       = errors.New("unauthorized")
)

const (
	SessionDuration = 24 * time.Hour * 30 // 30 days
	BcryptCost      = 12
)

type Auth struct {
	db *db.DB
}

func New(database *db.DB) *Auth {
	return &Auth{db: database}
}

func (a *Auth) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (a *Auth) CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (a *Auth) CreateUser(username, password, role string) (*db.User, error) {
	existing, _ := a.db.GetUserByUsername(username)
	if existing != nil {
		return nil, ErrUserExists
	}

	hash, err := a.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &db.User{
		Username:     username,
		PasswordHash: hash,
		Role:         role,
	}

	id, err := a.db.CreateUser(user)
	if err != nil {
		return nil, err
	}

	user.ID = id
	return user, nil
}

func (a *Auth) Authenticate(username, password string) (*db.User, error) {
	user, err := a.db.GetUserByUsername(username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !a.CheckPassword(user.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

func (a *Auth) CreateSession(userID int64) (*db.Session, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	session := &db.Session{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(SessionDuration),
	}

	if err := a.db.CreateSession(session); err != nil {
		return nil, err
	}

	return session, nil
}

func (a *Auth) ValidateSession(sessionID string) (*db.User, error) {
	session, err := a.db.GetSession(sessionID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	if time.Now().After(session.ExpiresAt) {
		a.db.DeleteSession(sessionID)
		return nil, ErrSessionExpired
	}

	user, err := a.db.GetUserByID(session.UserID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	return user, nil
}

func (a *Auth) DeleteSession(sessionID string) error {
	return a.db.DeleteSession(sessionID)
}

func (a *Auth) DeleteExpiredSessions() error {
	return a.db.DeleteExpiredSessions()
}

func (a *Auth) HasUsers() (bool, error) {
	count, err := a.db.CountUsers()
	return count > 0, err
}

func (a *Auth) GetUser(id int64) (*db.User, error) {
	return a.db.GetUserByID(id)
}

func (a *Auth) GetUsers() ([]db.User, error) {
	return a.db.GetUsers()
}

func (a *Auth) UpdateUserPassword(userID int64, newPassword string) error {
	hash, err := a.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return a.db.UpdateUserPassword(userID, hash)
}

func (a *Auth) UpdateUserRole(userID int64, role string) error {
	return a.db.UpdateUserRole(userID, role)
}

func (a *Auth) DeleteUser(userID int64) error {
	return a.db.DeleteUser(userID)
}

func generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
