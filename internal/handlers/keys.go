package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/mohit4bug/mo-sh/internal/models"
	"github.com/mohit4bug/mo-sh/internal/shared"
	"github.com/mohit4bug/mo-sh/pkg/db"
)

func FindAllKeys(c *gin.Context) {
	db := db.GetDB()

	rows, err := db.Query("SELECT id, name, is_external FROM keys")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var keys = make([]models.Key, 0)

	for rows.Next() {
		var key models.Key

		if err := rows.Scan(&key.ID, &key.Name, &key.IsExternal); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		keys = append(keys, key)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
		"data":    gin.H{"keys": keys},
	})
}

func FindKeyByID(c *gin.Context) {
	db := db.GetDB()
	id := c.Param("keyID")

	var key models.Key
	err := db.QueryRow("SELECT id, name, key, type FROM keys WHERE id = $1", id).Scan(&key.ID, &key.Name, &key.Key, &key.Type)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	publicKey, err := key.ExtractPublicKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
		"data": gin.H{"key": gin.H{
			"id":        key.ID,
			"name":      key.Name,
			"key":       key.Key,
			"type":      key.Type,
			"publicKey": publicKey,
		}},
	})
}

type GenerateKeyPairInput struct {
	Type string `json:"type" binding:"required"`
}

func GenerateKeyPair(c *gin.Context) {
	var input GenerateKeyPairInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Type != "rsa" && input.Type != "ed25519" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported key type"})
		return
	}

	dir, err := os.MkdirTemp("/tmp", "sshkey_*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer os.RemoveAll(dir)

	filename := shared.GenerateRandomString(10)
	privateKeyPath := filepath.Join(dir, fmt.Sprintf("id_%s_%s", input.Type, filename))
	publicKeyPath := privateKeyPath + ".pub"

	cmd := exec.Command("ssh-keygen", "-t", input.Type, "-C", filename, "-f", privateKeyPath, "-N", "")
	if input.Type == "rsa" {
		cmd.Args = append(cmd.Args, "-b", "4096")
	}

	if err := cmd.Run(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	privateKey, err := os.ReadFile(privateKeyPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	publicKey, err := os.ReadFile(publicKeyPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	copyCmd := fmt.Sprintf("echo %s >> ~/.ssh/authorized_keys", string(publicKey))

	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
		"data": gin.H{"key": gin.H{
			"privateKey": string(privateKey),
			"publicKey":  string(publicKey),
			"copyCmd":    copyCmd,
		}},
	})
}

type KeyInput struct {
	Name string `json:"name" binding:"required"`
	Key  string `json:"key" binding:"required"`
	Type string `json:"type" binding:"required"`
}

func CreateKey(c *gin.Context) {
	var input KeyInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Type != "rsa" && input.Type != "ed25519" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported key type"})
		return
	}

	db := db.GetDB()

	_, err := db.Exec(
		"INSERT INTO keys (name, key, type, is_external) VALUES ($1, $2, $3, $4)",
		input.Name, input.Key, input.Type, false,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "OK"})
}
