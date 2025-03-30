package workers

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mohit4bug/mo-sh/internal/models"
	"github.com/mohit4bug/mo-sh/pkg/ws"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/ssh"
)

type dockerInstallationWorker struct {
	DB            *sqlx.DB
	RedisClient   *redis.Client
	Ctx           *context.Context
	SocketManager *ws.WebSocketManager
}

func NewDockerInstallationWorker(db *sqlx.DB, redisClient *redis.Client, ctx *context.Context, socketManager *ws.WebSocketManager) *dockerInstallationWorker {
	return &dockerInstallationWorker{
		DB:            db,
		RedisClient:   redisClient,
		Ctx:           ctx,
		SocketManager: socketManager,
	}
}

func (w *dockerInstallationWorker) Start(numWorkers int) {
	for range numWorkers {
		go w.worker()
	}
}

func (w *dockerInstallationWorker) worker() {
	for {
		task, err := w.RedisClient.BRPopLPush(*w.Ctx, "docker_installation_queue", "processing_queue", 0).Result()
		if err != nil {
			continue
		}

		parts := strings.Split(task, ":")
		if len(parts) != 2 {
			continue
		}
		serverID, taskID := parts[0], parts[1]

		w.installDocker(serverID, taskID)
	}
}

func (w *dockerInstallationWorker) installDocker(serverID string, taskID string) {
	type ServerWithKey struct {
		models.Server
		Key string `db:"key"`
	}

	query := `
		select
			s.*, k.key
		from
			servers s
		inner join
			keys k ON s.key_id = k.id
		where
			s.id = $1
	`

	var server ServerWithKey
	if err := w.DB.GetContext(*w.Ctx, &server, query, serverID); err != nil {
		if err == sql.ErrNoRows {
			w.SocketManager.SendMessage(taskID, map[string]any{"error": "Server not found."})
		} else {
			w.SocketManager.SendMessage(taskID, map[string]any{"error": "Internal Server Error"})
		}
		w.SocketManager.RemoveClient(taskID)
		return
	}

	signer, err := ssh.ParsePrivateKey([]byte(server.Key))
	if err != nil {
		w.SocketManager.SendMessage(taskID, map[string]any{"error": "Internal Server Error"})
		w.SocketManager.RemoveClient(taskID)
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

	address := fmt.Sprintf("%s:%d", server.Hostname, server.Port)

	conn, err := ssh.Dial("tcp", address, config)
	if err != nil {
		w.SocketManager.SendMessage(taskID, map[string]any{"error": "Internal Server Error"})
		w.SocketManager.RemoveClient(taskID)
		return
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		w.SocketManager.SendMessage(taskID, map[string]any{"error": "Internal Server Error"})
		w.SocketManager.RemoveClient(taskID)
		return
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		w.SocketManager.SendMessage(taskID, map[string]any{"error": "Internal Server Error"})
		w.SocketManager.RemoveClient(taskID)
		return
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		w.SocketManager.SendMessage(taskID, map[string]any{"error": "Internal Server Error"})
		w.SocketManager.RemoveClient(taskID)
		return
	}

	cmd := `
		for pkg in docker.io docker-doc docker-compose docker-compose-v2 podman-docker containerd runc; do 
			sudo apt-get remove -y $pkg
		done

		sudo apt-get update
		sudo apt-get install -y ca-certificates curl

		# Set up Docker's official GPG key
		sudo install -m 0755 -d /etc/apt/keyrings
		sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
		sudo chmod a+r /etc/apt/keyrings/docker.asc

		# Add Docker's repository to Apt sources
		echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
		$(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}") stable" | \
		sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

		sudo apt-get update
		sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
	`
	if err := session.Start(cmd); err != nil {
		w.SocketManager.SendMessage(taskID, map[string]any{"error": "Internal Server Error"})
		w.SocketManager.RemoveClient(taskID)
		return
	}

	go w.streamLogs(taskID, stdout, "message")
	go w.streamLogs(taskID, stderr, "error")

	if err := session.Wait(); err != nil {
		w.SocketManager.SendMessage(taskID, map[string]any{"error": "Internal Server Error"})
		w.SocketManager.RemoveClient(taskID)
		return
	}

	w.SocketManager.SendMessage(taskID, map[string]any{"message": "Installation completed successfully."})
	w.SocketManager.RemoveClient(taskID)
}

func (w *dockerInstallationWorker) streamLogs(taskID string, reader io.Reader, eventType string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		w.SocketManager.SendMessage(taskID, map[string]any{eventType: scanner.Text()})
	}
	if err := scanner.Err(); err != nil {
		w.SocketManager.SendMessage(taskID, map[string]any{"error": "Internal Server Error"})
		w.SocketManager.RemoveClient(taskID)
		return
	}
}
