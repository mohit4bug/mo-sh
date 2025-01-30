package handlers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mohit4bug/mo-sh/c"
	"github.com/mohit4bug/mo-sh/db"
	"github.com/mohit4bug/mo-sh/models"
	"golang.org/x/crypto/ssh"
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

	var privateKey models.PrivateKey

	err := db.QueryRow("SELECT id, name FROM private_keys WHERE id = $1", id).Scan(&privateKey.ID, &privateKey.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSONResponse(w, http.StatusNotFound, c.JSON{
				"message": "Not Found",
			})
			return
		}
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
		return
	}

	c.JSONResponse(w, http.StatusOK, c.JSON{
		"message": "OK",
		"data": map[string]interface{}{
			"privateKey": privateKey,
		},
	})
}

// TODO: Implement support for ED25519 key type.
func GenerateKeyPair(w http.ResponseWriter, r *http.Request) {
	// TODO: Validate request body

	var body GenerateKeyPairBody
	if err := c.JSONParseRequestBody(w, r, &body); err != nil {
		return
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
		return
	}

	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		c.JSONResponse(w, http.StatusInternalServerError, c.JSON{
			"message": "Internal Server Error",
		})
		return
	}

	privatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	publicKey := ssh.MarshalAuthorizedKey(signer.PublicKey())

	c.JSONResponse(w, http.StatusOK, c.JSON{
		"message": "OK",
		"data": map[string]interface{}{
			"privateKey": string(privatePEM),
			"publicKey":  string(publicKey),
		},
	})

}

type GenerateKeyPairBody struct {
	Type string `json:"type"`
}
