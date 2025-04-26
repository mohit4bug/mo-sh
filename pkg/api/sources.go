package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/mohit4bug/mo-sh/internal/config"
	"github.com/mohit4bug/mo-sh/internal/models"
	"github.com/mohit4bug/mo-sh/internal/shared"
	"github.com/redis/go-redis/v9"
)

const githubAppStateTTL = 10 * time.Minute

type SourceRepository interface {
	Create(c *gin.Context)
	FindAll(c *gin.Context)
	FindByID(c *gin.Context)
	RegisterGithubApp(c *gin.Context)
}

type sourceRepository struct {
	DB          *sqlx.DB
	RedisClient *redis.Client
	Ctx         context.Context
	Config      config.Config
}

func NewSourceRepository(db *sqlx.DB, redisClient *redis.Client, ctx context.Context, cfg config.Config) *sourceRepository {
	return &sourceRepository{
		DB:          db,
		RedisClient: redisClient,
		Ctx:         ctx,
		Config:      cfg,
	}
}

func (r *sourceRepository) Create(c *gin.Context) {
	var input models.CreateSource
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newSource := &models.Source{
		Name: input.Name,
		Type: input.Type,
	}

	_, err := r.DB.ExecContext(
		r.Ctx,
		"insert into sources (name, type) values ($1, $2)",
		newSource.Name, newSource.Type,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "OK"})
}

func (r *sourceRepository) FindAll(c *gin.Context) {
	sources := []models.Source{}

	if err := r.DB.SelectContext(r.Ctx, &sources, "SELECT * FROM sources"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
		"data":    gin.H{"sources": sources},
	})
}

func (r *sourceRepository) FindByID(c *gin.Context) {
	sourceID := c.Param("sourceID")

	var source models.Source
	err := r.DB.GetContext(
		r.Ctx,
		&source,
		`select * from sources where id = $1`,
		sourceID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "404"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
		"data":    gin.H{"source": source},
	})
}

func (r *sourceRepository) RegisterGithubApp(c *gin.Context) {
	sourceID := c.Param("sourceID")

	var source models.Source
	if err := r.DB.GetContext(r.Ctx, &source, "select * from sources where id = $1", sourceID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Source not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	state := shared.GenerateRandomString(32)
	action := fmt.Sprintf("https://github.com/settings/apps/new?state=%s", state)
	redirectURL := r.Config.PublicURL + "/api/v1/webhooks/github/redirect"

	err := r.RedisClient.Set(r.Ctx, state, sourceID, githubAppStateTTL).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	manifest := gin.H{
		"name":         source.Name,
		"url":          r.Config.ClientURL,
		"redirect_url": redirectURL,
		"default_permissions": gin.H{
			"contents":       "read",
			"metadata":       "read",
			"emails":         "read",
			"administration": "read",
		},
	}

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tmpl, err := template.ParseFiles("templates/register_github_app.html")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = tmpl.Execute(c.Writer, gin.H{
		"Action":   action,
		"Manifest": string(manifestJSON),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
