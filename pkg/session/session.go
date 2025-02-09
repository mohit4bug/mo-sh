package session

import (
	"context"
	"time"

	"github.com/mohit4bug/mo-sh/internal/shared"
	"github.com/mohit4bug/mo-sh/pkg/rdb"
)

type Session struct {
	ID     string
	UserID string
}

const DefaultSessionTimeout = 7 * 24 * time.Hour

func CreateSession(userID string) (*Session, error) {
	rdb := rdb.GetRedis()
	ctx := context.Background()

	sessionID := shared.GenerateRandomString(32)
	session := &Session{
		ID:     sessionID,
		UserID: userID,
	}

	if err := rdb.Set(ctx, sessionID, userID, DefaultSessionTimeout).Err(); err != nil {
		return nil, err
	}

	return session, nil
}

func DeleteSession(sessionID string) error {
	rdb := rdb.GetRedis()
	ctx := context.Background()

	return rdb.Del(ctx, sessionID).Err()
}

func GetSession(sessionID string) (*Session, error) {
	rdb := rdb.GetRedis()
	ctx := context.Background()

	userID, err := rdb.Get(ctx, sessionID).Result()
	if err != nil {
		return nil, err
	}

	return &Session{
		ID:     sessionID,
		UserID: userID,
	}, nil
}
