package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mohit4bug/mo-sh/c"
	"github.com/mohit4bug/mo-sh/db"
	"github.com/mohit4bug/mo-sh/models"
	"github.com/mohit4bug/mo-sh/rdb"
)

func CreateSource(w http.ResponseWriter, r *http.Request) {
	// TODO: Validate request body

	var body CreateSourceBody
	c.JSONParseRequestBody(w, r, &body)

	db := db.GetDB()
	id := c.GenerateULID()

	_, err := db.Exec("INSERT INTO sources (id, name, type) VALUES ($1, $2, $3)", id, body.Name, body.Type)
	if err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
		return
	}

	c.JSONResponse(w, http.StatusCreated, c.JSON{
		"message": "OK",
		"data": map[string]interface{}{
			"id": id,
		},
	})
}

func FindAllSources(w http.ResponseWriter, r *http.Request) {
	db := db.GetDB()

	rows, err := db.Query("SELECT * FROM sources")
	if err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
		return
	}
	defer rows.Close()

	var sources []models.Source

	for rows.Next() {
		var source models.Source

		if err := rows.Scan(&source.ID, &source.Name, &source.Type); err != nil {
			c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
				"message": "Internal Server Error",
			})
			return
		}

		sources = append(sources, source)
	}

	c.JSONResponse(w, http.StatusOK, c.JSON{
		"message": "OK",
		"data": map[string]interface{}{
			"sources": sources,
		},
	})
}

func RegisterGithubApp(w http.ResponseWriter, r *http.Request) {
	sourceID := chi.URLParam(r, "sourceID")

	db := db.GetDB()

	var source models.Source
	err := db.QueryRow("SELECT name FROM sources WHERE id = $1", sourceID).Scan(&source.Name)
	if err == sql.ErrNoRows {
		c.JSONResponse(w, http.StatusNotFound, c.JSON{
			"message": "Not Found",
		})
		return
	} else if err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
		return
	}

	state := c.GenerateULID()
	action := "https://github.com/settings/apps/new?state=" + state

	rdb := rdb.GetRedisClient()
	ctx := context.Background()

	err = rdb.Set(ctx, state, sourceID, 0).Err()
	if err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
		return
	}

	manifest := c.JSON{
		"name":         source.Name,
		"url":          "https://example.com",
		"redirect_url": "http://localhost:3000/webhooks/github/redirect",
	}

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
		return
	}

	tmpl, err := template.ParseFiles("templates/register_github_app.html")
	if err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
		return
	}

	err = tmpl.Execute(w, map[string]interface{}{
		"Action":   action,
		"Manifest": string(manifestJSON),
	})
	if err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
	}
}

type CreateSourceBody struct {
	Name string `json:"name"`
	Type string `json:"type"`
}
