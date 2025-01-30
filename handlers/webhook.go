package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mohit4bug/mo-sh/c"
	"github.com/mohit4bug/mo-sh/db"
	"github.com/mohit4bug/mo-sh/models"
	"github.com/mohit4bug/mo-sh/rdb"
	"github.com/redis/go-redis/v9"
)

func HandleGithubRedirect(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	if state == "" || code == "" {
		c.JSONResponse(w, http.StatusBadRequest, c.JSON{
			"message": "Invalid request",
		})
		return
	}

	rdb := rdb.GetRedisClient()
	ctx := context.Background()

	sourceID, err := rdb.Get(ctx, state).Result()
	if err == redis.Nil {
		c.JSONResponse(w, http.StatusNotFound, c.JSON{
			"message": "Source not found",
		})
		return
	} else if err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
		return
	}

	url := fmt.Sprintf("https://api.github.com/app-manifests/%s/conversions", code)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
		return
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
		return
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated:
		var githubApp models.GithubApp
		if err := json.NewDecoder(resp.Body).Decode(&githubApp); err != nil {
			c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
				"message": "Internal Server Error",
			})
			return
		}

		ownerJSON, err := json.Marshal(githubApp.Owner)
		if err != nil {
			c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
				"message": "Internal Server Error",
			})
			return
		}

		permissionsJSON, err := json.Marshal(githubApp.Permissions)
		if err != nil {
			c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
				"message": "Internal Server Error",
			})
			return
		}

		eventsJSON, err := json.Marshal(githubApp.Events)
		if err != nil {
			c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
				"message": "Internal Server Error",
			})
			return
		}

		db := db.GetDB()
		tx, err := db.Begin()
		if err != nil {
			c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
				"message": "Internal Server Error",
			})
			return
		}
		defer tx.Rollback()

		privateKeyId := c.GenerateULID()
		_, err = tx.Exec(`INSERT INTO private_keys (id, name, key, type) VALUES ($1, $2, $3, $4)`, privateKeyId, fmt.Sprintf("gh-%s", githubApp.Name), githubApp.PEM, "rsa")
		if err != nil {
			c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
				"message": "Internal Server Error",
			})
			return
		}

		githubAppId := c.GenerateULID()
		_, err = tx.Exec(`
			INSERT INTO github_apps (
				id, 
				slug, 
				client_id,
				node_id, 
				owner, 
				name, 
				description, 
				external_url, 
				html_url, 
				created_at, 
				updated_at, 
				permissions, 
				events,
				source_id,
				client_secret,
				webhook_secret,
				private_key_id
			) 
			VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
			)
		`,
			githubAppId,
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
			privateKeyId,
		)
		if err != nil {
			c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
				"message": "Internal Server Error",
				"err":     err.Error(),
			})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
				"message": "Internal Server Error",
			})
			return
		}

		c.JSONResponse(w, http.StatusCreated, c.JSON{
			"message": "OK",
			"data": c.JSON{
				"id": privateKeyId,
			},
		})
		return
	case http.StatusNotFound:
		c.JSONResponse(w, http.StatusNotFound, c.JSON{
			"message": "Resource not found",
		})
		return
	case http.StatusUnprocessableEntity:
		c.JSONResponse(w, http.StatusUnprocessableEntity, c.JSON{
			"message": "Validation failed, or the endpoint has been spammed.",
		})
		return
	default:
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
	}
}
