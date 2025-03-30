package session

import (
	"context"
	"time"

	"github.com/mohit4bug/mo-sh/internal/shared"
	"github.com/redis/go-redis/v9"
)

type Session struct {
	ID     string
	UserID string
}

const DefaultSessionTimeout = 7 * 24 * time.Hour

type Store interface {
	Create(userID string) (*Session, error)
	Delete(sessionID string) error
	FindByID(sessionID string) (*Session, error)
}

type store struct {
	RedisClient *redis.Client
	Ctx         *context.Context
}

func NewSessionStore(redisClient *redis.Client, ctx *context.Context) *store {
	return &store{
		RedisClient: redisClient,
		Ctx:         ctx,
	}
}

func (s *store) Create(userID string) (*Session, error) {
	sessionID := shared.GenerateRandomString(32)
	session := &Session{
		ID:     sessionID,
		UserID: userID,
	}

	if err := s.RedisClient.Set(*s.Ctx, sessionID, userID, DefaultSessionTimeout).Err(); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *store) Delete(sessionID string) error {
	return s.RedisClient.Del(*s.Ctx, sessionID).Err()
}

func (s *store) FindByID(sessionID string) (*Session, error) {
	userID, err := s.RedisClient.Get(*s.Ctx, sessionID).Result()
	if err != nil {
		return nil, err
	}
	return &Session{
		ID:     sessionID,
		UserID: userID,
	}, nil
}
