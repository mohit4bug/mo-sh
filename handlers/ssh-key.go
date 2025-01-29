package handlers

import (
	"net/http"

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

func GenerateKeyPair(w http.ResponseWriter, r *http.Request) {}
