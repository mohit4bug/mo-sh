package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/mohit4bug/mo-sh/c"
	"github.com/mohit4bug/mo-sh/db"
	"github.com/mohit4bug/mo-sh/models"
)

func FindAllSSHKeys(w http.ResponseWriter, r *http.Request) {
	db := db.GetDB()

	rows, err := db.Query("SELECT id, name, is_external FROM private_keys")
	if err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
		return
	}
	defer rows.Close()

	var privateKeys []models.PrivateKey

	for rows.Next() {
		var privateKey models.PrivateKey

		if err := rows.Scan(&privateKey.ID, &privateKey.Name, &privateKey.IsExternal); err != nil {
			c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
				"message": "Internal Server Error",
			})
			return
		}

		privateKeys = append(privateKeys, privateKey)
	}

	c.JSONResponse(w, http.StatusOK, c.JSON{
		"message": "OK",
		"data": map[string]interface{}{
			"privateKeys": privateKeys,
		},
	})
}

func FindSSHKeyByID(w http.ResponseWriter, r *http.Request) {
	db := db.GetDB()
	id := chi.URLParam(r, "sshKeyID")

	type SSHKey struct {
		models.PrivateKey
		PublicKey string `json:"publicKey"`
	}

	var sshKey SSHKey

	err := db.QueryRow("SELECT id, name, key, type FROM private_keys WHERE id = $1", id).Scan(&sshKey.ID, &sshKey.Name, &sshKey.Key, &sshKey.Type)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSONResponse(w, http.StatusNotFound, c.JSON{
				"message": "Not Found",
			})
			return
		}
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
			"error":   err.Error(),
		})
		return
	}

	sshKey.PublicKey, err = c.ExtractPublicKey(*sshKey.Key, *sshKey.Type)
	if err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
		return
	}

	c.JSONResponse(w, http.StatusOK, c.JSON{
		"message": "OK",
		"data": c.JSON{
			"sshKey": sshKey,
		},
	})
}

func GenerateKeyPair(w http.ResponseWriter, r *http.Request) {
	var body GenerateKeyPairBody
	if err := c.JSONParseRequestBody(w, r, &body); err != nil {
		return
	}

	if body.Type != "rsa" && body.Type != "ed25519" {
		c.JSONResponse(w, http.StatusBadRequest, c.JSON{
			"message": "Unsupported key type",
		})
		return
	}

	dir, err := os.MkdirTemp("/tmp", "sshkey_*")
	if err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
		return
	}
	defer os.RemoveAll(dir)

	filename := c.GenerateULID()
	privateKeyPath := filepath.Join(dir, fmt.Sprintf("id_%s_%s", body.Type, filename))
	publicKeyPath := privateKeyPath + ".pub"

	cmd := exec.Command("ssh-keygen", "-t", body.Type, "-C", filename, "-f", privateKeyPath, "-N", "")
	if body.Type == "rsa" {
		cmd.Args = append(cmd.Args, "-b", "4096")
	}

	if err := cmd.Run(); err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
		return
	}

	privateKey, err := os.ReadFile(privateKeyPath)
	if err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
		return
	}

	publicKey, err := os.ReadFile(publicKeyPath)
	if err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
		return
	}

	copyCmd := fmt.Sprintf("echo %s >> ~/.ssh/authorized_keys", string(publicKey))

	c.JSONResponse(w, http.StatusOK, c.JSON{
		"message": "OK",
		"data": c.JSON{
			"privateKey": string(privateKey),
			"publicKey":  string(publicKey),
			"copyCmd":    copyCmd,
		},
	})
}

func CreateSSHKey(w http.ResponseWriter, r *http.Request) {
	var body CreateSSHKeyBody
	if err := c.JSONParseRequestBody(w, r, &body); err != nil {
		return
	}

	if body.Name == "" {
		c.JSONResponse(w, http.StatusBadRequest, c.JSON{
			"message": "Name is required",
		})
		return
	}

	if body.Type != "rsa" && body.Type != "ed25519" {
		c.JSONResponse(w, http.StatusBadRequest, c.JSON{
			"message": "Unsupported key type",
		})
		return
	}

	db := db.GetDB()
	id := c.GenerateULID()

	_, err := db.Exec(
		"INSERT INTO private_keys (id, name, key, type, is_external) VALUES ($1, $2, $3, $4, $5)",
		id, body.Name, body.Key, body.Type, false,
	)
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

type GenerateKeyPairBody struct {
	Type string `json:"type"`
}

type CreateSSHKeyBody struct {
	Name string `json:"name"`
	Key  string `json:"key"`
	Type string `json:"type"`
}
