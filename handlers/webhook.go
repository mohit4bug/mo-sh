package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mohit4bug/mo-sh/c"
	"github.com/mohit4bug/mo-sh/db"
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

	rdbClient := rdb.GetRedisClient()
	ctx := context.Background()

	sourceID, err := rdbClient.Get(ctx, state).Result()
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
		var githubApp GithubApp
		if err := json.NewDecoder(resp.Body).Decode(&githubApp); err != nil {
			c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
				"message": "Internal Server Error",
			})
			return
		}

		dbClient := db.GetDB()
		id := c.GenerateULID()

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

		_, err = dbClient.Exec(`
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
				source_id
			) 
			VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
			)
		`,
			id,
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
		)
		if err != nil {
			c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
				"message": "Internal Server Error",
				"err":     err.Error(),
			})
			return
		}

		c.JSONResponse(w, http.StatusCreated, c.JSON{
			"message": "OK",
			"data": map[string]interface{}{
				"id": id,
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

type Owner struct {
	Login             string `json:"login"`
	ID                int    `json:"id"`
	NodeID            string `json:"node_id"`
	AvatarURL         string `json:"avatar_url"`
	GravatarID        string `json:"gravatar_id"`
	URL               string `json:"url"`
	HTMLURL           string `json:"html_url"`
	FollowersURL      string `json:"followers_url"`
	FollowingURL      string `json:"following_url"`
	GistsURL          string `json:"gists_url"`
	StarredURL        string `json:"starred_url"`
	SubscriptionsURL  string `json:"subscriptions_url"`
	OrganizationsURL  string `json:"organizations_url"`
	ReposURL          string `json:"repos_url"`
	EventsURL         string `json:"events_url"`
	ReceivedEventsURL string `json:"received_events_url"`
	Type              string `json:"type"`
	SiteAdmin         bool   `json:"site_admin"`
}

type Permissions map[string]string

type GithubApp struct {
	ID          int64       `json:"id"`
	Slug        string      `json:"slug"`
	ClientID    string      `json:"client_id"`
	NodeID      string      `json:"node_id"`
	Owner       Owner       `json:"owner"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	ExternalURL string      `json:"external_url"`
	HTMLURL     string      `json:"html_url"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	Permissions Permissions `json:"permissions"`
	Events      []string    `json:"events"`
}
