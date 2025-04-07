package api

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/mohit4bug/mo-sh/internal/models"
	"github.com/mohit4bug/mo-sh/pkg/session"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type UserRepository interface {
	Register(c *gin.Context)
	Login(c *gin.Context)
}

type userRepository struct {
	DB          *sqlx.DB
	RedisClient *redis.Client
	Ctx         context.Context
}

func NewUserRepository(db *sqlx.DB, redisClient *redis.Client, ctx context.Context) *userRepository {
	return &userRepository{
		DB:          db,
		RedisClient: redisClient,
		Ctx:         ctx,
	}
}

func (r *userRepository) Register(c *gin.Context) {
	var input models.RegisterUser
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var exists bool
	if err := r.DB.QueryRowContext(r.Ctx, "select exists(select 1 from users where email = $1)", input.Email).Scan(&exists); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email already exists"})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	newUser := &models.User{
		Email:        input.Email,
		PasswordHash: string(passwordHash),
	}

	if err := r.DB.QueryRowContext(
		r.Ctx,
		"insert into users (email, password_hash) values ($1, $2)",
		newUser.Email,
		newUser.PasswordHash,
	).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OK"})
}

func (r *userRepository) Login(c *gin.Context) {
	var input models.LoginUser
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := r.DB.QueryRowContext(
		r.Ctx,
		"select id, email, password_hash from users where email = $1",
		input.Email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	sessionDuration := session.DefaultSessionTimeout // Keep session and cookie duration same.
	sessionStore := session.NewSessionStore(r.RedisClient, r.Ctx)
	session, err := sessionStore.Create(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("session_id", session.ID, int(sessionDuration.Seconds()), "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
		"data": gin.H{"user": gin.H{
			"id": user.ID,
		}},
	})
}
