package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/mohit4bug/mo-sh/internal/models"
	"github.com/mohit4bug/mo-sh/internal/shared"
	"github.com/redis/go-redis/v9"
)

type KeyRepository interface {
	Create(c *gin.Context)
	FindAll(c *gin.Context)
	FindByID(c *gin.Context)
	GenerateKey(c *gin.Context)
}

type keyRepository struct {
	DB          *sqlx.DB
	RedisClient *redis.Client
	Ctx         context.Context
}

func NewKeyRepository(db *sqlx.DB, redisClient *redis.Client, ctx context.Context) *keyRepository {
	return &keyRepository{
		DB:          db,
		RedisClient: redisClient,
		Ctx:         ctx,
	}
}

func (r *keyRepository) Create(c *gin.Context) {
	var input models.CreateKey
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newKey := &models.Key{
		Name:       input.Name,
		Key:        input.Key,
		IsExternal: false,
	}

	query := "INSERT INTO keys (name, key, is_external) VALUES (:name, :key, :is_external)"
	_, err := r.DB.NamedExecContext(r.Ctx, query, newKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "OK"})
}

func (r *keyRepository) FindAll(c *gin.Context) {
	var keys []models.Key = []models.Key{}
	if err := r.DB.SelectContext(r.Ctx, &keys, "SELECT * FROM keys"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
		"data":    gin.H{"keys": keys},
	})
}

func (r *keyRepository) FindByID(c *gin.Context) {
	keyID := c.Param("keyID")

	var key models.Key
	if err := r.DB.GetContext(r.Ctx, &key, "select * from keys where id = $1", keyID); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "404"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
		"data":    gin.H{"key": key},
	})
}

func (r *keyRepository) GenerateKey(c *gin.Context) {
	var input models.GenerateKey
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Type != "rsa" && input.Type != "ed25519" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported key type"})
		return
	}

	dir, err := os.MkdirTemp("/tmp", "sshkey_*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer os.RemoveAll(dir)

	filename := shared.GenerateRandomString(10)
	privateKeyPath := filepath.Join(dir, fmt.Sprintf("id_%s_%s", input.Type, filename))
	publicKeyPath := privateKeyPath + ".pub"

	cmd := exec.CommandContext(r.Ctx, "ssh-keygen", "-t", input.Type, "-C", filename, "-f", privateKeyPath, "-N", "")
	if input.Type == "rsa" {
		cmd.Args = append(cmd.Args, "-b", "4096")
	}

	if err := cmd.Run(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	privateKey, err := os.ReadFile(privateKeyPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	publicKey, err := os.ReadFile(publicKeyPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	copyCmd := fmt.Sprintf("echo %s >> ~/.ssh/authorized_keys", string(publicKey))

	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
		"data": gin.H{"key": gin.H{
			"privateKey": string(privateKey),
			"publicKey":  string(publicKey),
			"copyCmd":    copyCmd,
		}},
	})
}
