package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/mohit4bug/mo-sh/internal/models"
	"github.com/redis/go-redis/v9"
)

type WebhookRepository interface {
	HandleGithubRedirect(c *gin.Context)
}

type webhookRepository struct {
	DB          *sqlx.DB
	RedisClient *redis.Client
	Ctx         context.Context
}

func NewWebhookRepository(db *sqlx.DB, redisClient *redis.Client, ctx context.Context) *webhookRepository {
	return &webhookRepository{
		DB:          db,
		RedisClient: redisClient,
		Ctx:         ctx,
	}
}

func (r *webhookRepository) HandleGithubRedirect(c *gin.Context) {
	state := c.Query("state")
	code := c.Query("code")

	if state == "" || code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}

	sourceID, err := r.RedisClient.Get(r.Ctx, state).Result()
	if err != nil {
		if err == redis.Nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err = r.RedisClient.Del(r.Ctx, state).Err(); err != nil {
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
	req.Header.Set("User-Agent", "Mo-SH")

	client := &http.Client{Timeout: 10 * time.Second}
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

		ownerJSON, err := json.Marshal(githubAppResponse.Owner)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		permissionsJSON, err := json.Marshal(githubAppResponse.Permissions)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		eventsJSON, err := json.Marshal(githubAppResponse.Events)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		tx, err := r.DB.BeginTx(r.Ctx, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer tx.Rollback()

		var keyID string
		if err = tx.QueryRowContext(
			r.Ctx,
			`INSERT INTO keys (name, key, is_external) VALUES ($1, $2, $3) RETURNING id`,
			fmt.Sprintf("gh-%s", githubAppResponse.Name),
			githubAppResponse.PEM,
			true,
		).Scan(&keyID); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		_, err = tx.ExecContext(
			r.Ctx,
			`INSERT INTO github_apps (
				slug, client_id, node_id, owner, name, description, external_url, 
				html_url, created_at, updated_at, permissions, events, source_id, 
				client_secret, webhook_secret, key_id
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
			)`,
			githubAppResponse.Slug,
			githubAppResponse.ClientID,
			githubAppResponse.NodeID,
			ownerJSON,
			githubAppResponse.Name,
			githubAppResponse.Description,
			githubAppResponse.ExternalURL,
			githubAppResponse.HTMLURL,
			githubAppResponse.CreatedAt,
			githubAppResponse.UpdatedAt,
			permissionsJSON,
			eventsJSON,
			sourceID,
			githubAppResponse.ClientSecret,
			githubAppResponse.WebhookSecret,
			keyID,
		)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		_, err = tx.ExecContext(
			r.Ctx,
			`update sources set has_github_app = true where id = $1`,
			sourceID,
		)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if err = tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		clientURL := "http://localhost:3000/sources/" + sourceID
		c.Redirect(http.StatusFound, clientURL)
		return
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
}
