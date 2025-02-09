package handlers

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mohit4bug/mo-sh/internal/models"
	"github.com/mohit4bug/mo-sh/pkg/db"
	"github.com/mohit4bug/mo-sh/pkg/rdb"
	"golang.org/x/crypto/ssh"
)

type ServerInput struct {
	Name     string `json:"name" binding:"required"`
	Hostname string `json:"hostname" binding:"required"`
	Port     int    `json:"port" binding:"required"`
	KeyID    string `json:"keyId" binding:"required"`
}

func CreateServer(c *gin.Context) {
	var input ServerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := db.GetDB()
	var id string
	err := db.QueryRow("INSERT INTO servers (name, hostname, port, key_id) VALUES ($1, $2, $3, $4) RETURNING id", input.Name, input.Hostname, input.Port, input.KeyID).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "OK",
		"data": gin.H{"server": gin.H{
			"id": id,
		}},
	})
}

func FindAllServers(c *gin.Context) {
	db := db.GetDB()

	rows, err := db.Query("SELECT id, name, hostname, port, key_id FROM servers")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	servers := make([]models.Server, 0)

	for rows.Next() {
		var server models.Server
		if err := rows.Scan(&server.ID, &server.Name, &server.Hostname, &server.Port, &server.KeyID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		servers = append(servers, server)
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"servers": servers}})
}

func QueueDockerInstall(c *gin.Context) {
	serverID := c.Param("serverID")
	db := db.GetDB()

	// Verify if the installation status is currently running in the PostgreSQL table. If true, return.
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) 
		FROM docker_installations 
		WHERE server_id = $1 AND status = 'in_progress'
	`, serverID).Scan(&count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "An installation is already in progress"})
		return
	}

	// Verify SSH connection to the server.
	var (
		hostname, privateKey string
		port                 int
	)

	err = db.QueryRow(`
		SELECT s.hostname, s.port, k.key
		FROM servers AS s
		INNER JOIN keys AS k ON s.key_id = k.id
		WHERE s.id = $1
	`, serverID).Scan(&hostname, &port, &privateKey)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	signer, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	address := fmt.Sprintf("%s:%d", hostname, port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer client.Close()

	// Enqueue serverId into the docker_installation queue.
	rdb := rdb.GetRedis()
	ctx := context.Background()
	channel := "docker_installation" // 🐳

	err = rdb.Publish(ctx, channel, serverID).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Docker installation queued"})
}

func ListenForDockerInstall() {
	rdb := rdb.GetRedis()
	ctx := context.Background()
	channel := "docker_installation" // 🐳

	sub := rdb.Subscribe(ctx, channel)
	defer sub.Close()

	ch := sub.Channel()

	for msg := range ch {
		serverID := msg.Payload
		installDocker(serverID)
	}
}

func installDocker(serverID string) {
	db := db.GetDB()

	var (
		hostname, privateKey string
		port                 int
	)

	err := db.QueryRow(`
		SELECT s.hostname, s.port, k.key
		FROM servers AS s
		INNER JOIN keys AS k ON s.key_id = k.id
		WHERE s.id = $1
	`, serverID).Scan(&hostname, &port, &privateKey)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("Server not found")
			return
		}
		log.Println(err)
		return
	}

	signer, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		log.Println(err)
		return
	}

	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	address := fmt.Sprintf("%s:%d", hostname, port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		log.Println(err)
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		log.Println(err)
		return
	}
	defer session.Close()

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		log.Println(err)
		return
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		log.Println(err)
		return
	}

	cmd := `
		for pkg in docker.io docker-doc docker-compose docker-compose-v2 podman-docker containerd runc; do
			sudo apt-get remove $pkg
		done

		sudo apt-get update
		sudo apt-get install ca-certificates curl
		sudo install -m 0755 -d /etc/apt/keyrings
		sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
		sudo chmod a+r /etc/apt/keyrings/docker.asc

		echo \
			"deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
			$(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}") stable" | \
			sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

		sudo apt-get update

		sudo apt-get install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
`

	// Connection successful, adding installation record.
	_, err = db.Exec(`
		INSERT INTO docker_installations (server_id, status, logs)
		VALUES ($1, 'in_progress', '[]')
	`, serverID)
	if err != nil {
		log.Println(err)
		return
	}

	// Start the command
	if err := session.Start(cmd); err != nil {
		log.Println(err)
		return
	}

	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			saveDockerInstallationLogs(serverID, scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			saveDockerInstallationLogs(serverID, scanner.Text())
		}
	}()

	if err := session.Wait(); err != nil {
		log.Println(err)
		return
	}

	// Installation successful, updating installation record.
	_, err = db.Exec(`
		UPDATE docker_installations
		SET status = 'success'
		WHERE server_id = $1 AND status = 'in_progress'
	`, serverID)
	if err != nil {
		log.Println(err)
		return
	}
}

func saveDockerInstallationLogs(serverID string, log string) {
	db := db.GetDB()

	_, err := db.Exec(`
		UPDATE docker_installations
		SET logs = logs || TO_JSONB($1::TEXT)
		WHERE server_id = $2 AND status = 'in_progress'
	`, log, serverID)

	if err != nil {
		fmt.Println(err)
		return
	}
}
