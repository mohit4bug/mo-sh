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
	"github.com/mohit4bug/mo-sh/internal/shared"
	"github.com/mohit4bug/mo-sh/pkg/db"
	"github.com/mohit4bug/mo-sh/pkg/rdb"
)

type SourceInput struct {
	Name string `json:"name" binding:"required"`
	Type string `json:"type" binding:"required"`
}

func CreateSource(c *gin.Context) {
	var input SourceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := db.GetDB()
	var id string
	err := db.QueryRow("INSERT INTO sources (name, type) VALUES ($1, $2) RETURNING id", input.Name, input.Type).Scan(&id)
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

func FindAllSources(c *gin.Context) {
	db := db.GetDB()

	rows, err := db.Query("SELECT id, name, type FROM sources")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	sources := make([]models.Source, 0)

	for rows.Next() {
		var source models.Source
		if err := rows.Scan(&source.ID, &source.Name, &source.Type); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		sources = append(sources, source)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
		"data":    gin.H{"sources": sources},
	})
}

func RegisterGithubApp(c *gin.Context) {
	sourceID := c.Param("sourceID")
	db := db.GetDB()

	var sourceName string
	err := db.QueryRow("SELECT name FROM sources WHERE id = $1", sourceID).Scan(&sourceName)
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
		"name":         sourceName,
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
	}

	err = tmpl.Execute(c.Writer, gin.H{
		"Action":   action,
		"Manifest": string(manifestJSON),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
