package handlers

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/mohit4bug/mo-sh/internal/models"
	"github.com/mohit4bug/mo-sh/internal/repositories"
	"github.com/mohit4bug/mo-sh/internal/shared"
)

type KeyHandler struct {
	repo repositories.KeyRepository
}

func NewKeyHandler(repo repositories.KeyRepository) *KeyHandler {
	return &KeyHandler{repo: repo}
}

func (h *KeyHandler) FindAll(c *gin.Context) {
	keys, err := h.repo.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
		"data":    gin.H{"keys": keys},
	})
}

func (h *KeyHandler) FindByID(c *gin.Context) {
	id := c.Param("keyID")

	key, err := h.repo.FindByID(id)
	if err != nil {
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

type KeyInput struct {
	Name string `json:"name" binding:"required"`
	Key  string `json:"key" binding:"required"`
	Type string `json:"type" binding:"required"`
}

func (h *KeyHandler) Create(c *gin.Context) {
	var input KeyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Type != "rsa" && input.Type != "ed25519" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported key type"})
		return
	}

	key := &models.Key{
		Name: input.Name,
		Key:  input.Key,
		Type: input.Type,
	}

	if err := h.repo.Create(key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "OK"})
}

type GenerateKeyPairInput struct {
	Type string `json:"type" binding:"required"`
}

func (h *KeyHandler) GenerateKeyPair(c *gin.Context) {
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
