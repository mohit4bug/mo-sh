package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/gin-gonic/gin"
	"github.com/mohit4bug/mo-sh/internal/models"
	"github.com/mohit4bug/mo-sh/internal/repositories"
	"github.com/mohit4bug/mo-sh/internal/shared"
	"github.com/mohit4bug/mo-sh/pkg/rdb"
)

type SourceHandler struct {
	repo repositories.SourceRepository
}

func NewSourceHandler(repo repositories.SourceRepository) *SourceHandler {
	return &SourceHandler{repo: repo}
}

type SourceInput struct {
	Name string `json:"name" binding:"required"`
	Type string `json:"type" binding:"required"`
}

func (h *SourceHandler) Create(c *gin.Context) {
	var input SourceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	source := &models.Source{
		Name: input.Name,
		Type: input.Type,
	}

	id, err := h.repo.Create(source)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "OK",
		"data": gin.H{"source": gin.H{
			"id": id,
		}},
	})
}

func (h *SourceHandler) FindAll(c *gin.Context) {
	sources, err := h.repo.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
		"data":    gin.H{"sources": sources},
	})
}

func (h *SourceHandler) RegisterGithubApp(c *gin.Context) {
	sourceID := c.Param("sourceID")

	source, err := h.repo.FindByID(sourceID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	state := shared.GenerateRandomString(32)
	action := fmt.Sprintf("https://github.com/settings/apps/new?state=%s", state)

	rdb := rdb.GetRedis()
	ctx := context.Background()

	err = rdb.Set(ctx, state, sourceID, 0).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	manifest := gin.H{
		"name":         source.Name,
		"url":          "https://example.com",
		"redirect_url": "http://localhost:8080/api/webhooks/github/redirect",
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
