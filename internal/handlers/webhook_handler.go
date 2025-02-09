package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mohit4bug/mo-sh/internal/models"
	"github.com/mohit4bug/mo-sh/internal/repositories"
	"github.com/mohit4bug/mo-sh/pkg/rdb"
	"github.com/redis/go-redis/v9"
)

type WebhookHandler struct {
	repo repositories.WebhookRepository
}

func NewWebhookHandler(repo repositories.WebhookRepository) *WebhookHandler {
	return &WebhookHandler{repo: repo}
}

func (h *WebhookHandler) HandleGithubRedirect(c *gin.Context) {
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
		var githubAppResponse models.GithubAppResponse
		if err := json.NewDecoder(resp.Body).Decode(&githubAppResponse); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if err := h.repo.SaveGithubApp(&githubAppResponse, sourceID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "OK"})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
}
