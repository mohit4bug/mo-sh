package handlers

import (
	"net/http"

	"github.com/mohit4bug/mo-sh/c"
	"github.com/mohit4bug/mo-sh/db"
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

	var sources []Source

	for rows.Next() {
		var source Source

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

type CreateSourceBody struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Source struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}
