package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mohit4bug/mo-sh/c"
	"github.com/mohit4bug/mo-sh/db"
	"github.com/mohit4bug/mo-sh/models"
	"golang.org/x/crypto/ssh"
)

func CreateServer(w http.ResponseWriter, r *http.Request) {
	// TODO: Validate request body

	var body CreateServerBody
	err := c.JSONParseRequestBody(w, r, &body)
	if err != nil {
		return
	}

	db := db.GetDB()
	id := c.GenerateULID()

	_, err = db.Exec("INSERT INTO servers (id, name, hostname, port, private_key_id) VALUES ($1, $2, $3, $4, $5)", id, body.Name, body.Hostname, body.Port, body.PrivateKeyId)
	if err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
		return
	}

	c.JSONResponse(w, http.StatusCreated, c.JSON{
		"message": "OK",
		"data": c.JSON{
			"id": id,
		},
	})
}

func FindAllServers(w http.ResponseWriter, r *http.Request) {
	db := db.GetDB()

	rows, err := db.Query("SELECT id, name, hostname, port FROM servers")
	if err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
		return
	}
	defer rows.Close()

	var servers []models.Server

	for rows.Next() {
		var server models.Server

		if err := rows.Scan(&server.ID, &server.Name, &server.Hostname, &server.Port); err != nil {
			c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
				"message": "Internal Server Error",
			})
			return
		}

		servers = append(servers, server)
	}

	c.JSONResponse(w, http.StatusOK, c.JSON{
		"message": "OK",
		"data": c.JSON{
			"servers": servers,
		},
	})
}

func ValidateServer(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "serverID")
	db := db.GetDB()

	var hostname, privateKey string
	var port int

	err := db.QueryRow(`
	    SELECT s.hostname, s.port, pk.key
	    FROM servers AS s
	    INNER JOIN private_keys AS pk ON s.private_key_id = pk.id
	    WHERE s.id = $1
	`, serverID).Scan(&hostname, &port, &privateKey)
	if err == sql.ErrNoRows {
		c.JSONResponse(w, http.StatusNotFound, c.JSON{
			"message": "Server not found",
		})
		return
	} else if err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
		return
	}

	signer, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
		return
	}

	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // NOTE: Change this in production.
		Timeout:         5 * time.Second,
	}

	address := fmt.Sprintf("%s:%d", hostname, port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
			"error":   err.Error(),
		})
		return
	}
	defer client.Close()

	c.JSONResponse(w, http.StatusOK, c.JSON{
		"message": "OK",
	})
}

type CreateServerBody struct {
	Name         string `json:"name"`
	Hostname     string `json:"hostname"`
	Port         int    `json:"port"`
	PrivateKeyId string `json:"privateKeyId"`
}
