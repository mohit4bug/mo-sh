package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mohit4bug/mo-sh/internal/models"
	"github.com/mohit4bug/mo-sh/pkg/db"
	"github.com/mohit4bug/mo-sh/pkg/rdb"
	"github.com/redis/go-redis/v9"
)

func HandleGithubRedirect(c *gin.Context) {
	state := c.Query("state")
	code := c.Query("code")

	if state == "" || code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}

	rdb := rdb.GetRedis()
	ctx := context.Background()

	sourceID, err := rdb.Get(ctx, state).Result()
	if err != nil {
		if err == redis.Nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	url := fmt.Sprintf("https://api.github.com/app-manifests/%s/conversions", code)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
		return
	case http.StatusUnprocessableEntity:
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Validation failed, or the endpoint has been spammed."})
		return
	case http.StatusCreated:
		var githubApp models.GithubApp
		if err := json.NewDecoder(resp.Body).Decode(&githubApp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ownerJSON, err := json.Marshal(githubApp.Owner)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		permissionsJSON, err := json.Marshal(githubApp.Permissions)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		eventsJSON, err := json.Marshal(githubApp.Events)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		db := db.GetDB()
		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer tx.Rollback()

		var keyId string
		err = tx.QueryRow(
			`INSERT INTO keys (name, key, type) VALUES ($1, $2, $3) RETURNING id`,
			fmt.Sprintf("gh-%s", githubApp.Name),
			githubApp.PEM,
			"rsa",
		).Scan(&keyId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		_, err = tx.Exec(
			`INSERT INTO github_apps (
				slug, client_id, node_id, owner, name, description, external_url, 
				html_url, created_at, updated_at, permissions, events, source_id, 
				client_secret, webhook_secret, key_id
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
			)`,
			githubApp.Slug,
			githubApp.ClientID,
			githubApp.NodeID,
			ownerJSON,
			githubApp.Name,
			githubApp.Description,
			githubApp.ExternalURL,
			githubApp.HTMLURL,
			githubApp.CreatedAt,
			githubApp.UpdatedAt,
			permissionsJSON,
			eventsJSON,
			sourceID,
			githubApp.ClientSecret,
			githubApp.WebhookSecret,
			keyId,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "OK"})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
}
