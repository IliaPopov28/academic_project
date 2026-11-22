package user

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidInput = errors.New("invalid input")
var ErrUnauthenticated = errors.New("unauthenticated")

type Service struct {
	repo      *Repository
	secret    []byte
	dummyHash []byte
	ttl       time.Duration
}

func NewService(repo *Repository, secret []byte) (*Service, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("generate dummy bytes: %w", err)
	}

	dummyHash, err := bcrypt.GenerateFromPassword(buf, bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("generate dummy password hash: %w", err)
	}

	return &Service{
		repo:      repo,
		secret:    secret,
		dummyHash: dummyHash,
		ttl:       24 * time.Hour,
	}, nil
}

func (s *Service) Register(ctx context.Context, email, password, name string) (uuid.UUID, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)
	if email == "" || !strings.Contains(email, "@") || len(password) < 8 || name == "" {
		return uuid.Nil, ErrInvalidInput
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return uuid.Nil, fmt.Errorf("generate password hash: %w", err)
	}

	id := uuid.New()
	err = s.repo.Create(ctx, User{
		ID:           id,
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
	})
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			return uuid.UUID{}, ErrEmailTaken
		}
		return uuid.Nil, fmt.Errorf("create user: %w", err)
	}

	return id, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return "", ErrInvalidInput
	}

	u, err := s.repo.FindByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		return "", ErrUnauthenticated
	}
	if err != nil {
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", ErrUnauthenticated
	}

	return s.sign(u.ID)
}
func (s *Service) sign(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}
func (s *Service) ValidateToken(token string) (uuid.UUID, bool) {
	parsed, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil || !parsed.Valid {
		return uuid.Nil, false
	}
	claims, ok := parsed.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}
