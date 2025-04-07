package workers

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/mohit4bug/mo-sh/internal/models"
	"github.com/mohit4bug/mo-sh/internal/shared"
	"github.com/mohit4bug/mo-sh/pkg/ssh"
	"github.com/redis/go-redis/v9"
)

const (
	DockerPendingQueue    = "docker_installation:pending"
	DockerProcessingQueue = "docker_installation:processing"
)

type dockerInstallationWorker struct {
	DB          *sqlx.DB
	RedisClient *redis.Client
	Ctx         context.Context
	LogBuf      map[string]models.DockerInstallationLogs
	LogMu       sync.Mutex
}

func NewDockerInstallationWorker(db *sqlx.DB, redisClient *redis.Client, ctx context.Context) *dockerInstallationWorker {
	return &dockerInstallationWorker{
		DB:          db,
		RedisClient: redisClient,
		Ctx:         ctx,
		LogBuf:      make(map[string]models.DockerInstallationLogs),
		LogMu:       sync.Mutex{},
	}
}

func (w *dockerInstallationWorker) Start(numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		go w.worker()
	}

	// background task to flush logs periodically
	go w.flushLogs()
}

func (w *dockerInstallationWorker) worker() {
	for {
		serverID, err := w.RedisClient.BRPopLPush(w.Ctx, DockerPendingQueue, DockerProcessingQueue, 0).Result()
		if err != nil {
			continue
		}

		w.installDocker(serverID)

		// Delete the task from the processing queue.
		_, err = w.RedisClient.LRem(w.Ctx, DockerProcessingQueue, 1, serverID).Result()
		if err != nil {
			continue
		}
	}
}

func (w *dockerInstallationWorker) installDocker(serverID string) {
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
	if err := w.DB.GetContext(w.Ctx, &server, query, serverID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.appendLogToBuffer(serverID, models.DockerInstallationLog{
				Type:    models.LogTypeError,
				Content: shared.ErrNotFound,
			})
		} else {
			w.appendLogToBuffer(serverID, models.DockerInstallationLog{
				Type:    models.LogTypeError,
				Content: shared.ErrInternalServer,
			})
		}
		return
	}

	sshClient := ssh.NewClient(server.Hostname, server.Port, "root", []byte(server.Key))
	if err := sshClient.Connect(); err != nil {
		w.appendLogToBuffer(serverID, models.DockerInstallationLog{
			Type:    models.LogTypeError,
			Content: shared.ErrSSHConnection,
		})
		return
	}
	defer sshClient.Close()

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

	err := sshClient.ExecuteWithStreams(cmd,
		func(text string) {
			w.appendLogToBuffer(serverID, models.DockerInstallationLog{
				Type:    models.LogTypeInfo,
				Content: text,
			})
		},
		func(text string) {
			w.appendLogToBuffer(serverID, models.DockerInstallationLog{
				Type:    models.LogTypeError,
				Content: text,
			})
		},
	)

	if err != nil {
		w.appendLogToBuffer(serverID, models.DockerInstallationLog{
			Type:    models.LogTypeError,
			Content: shared.ErrInternalServer,
		})
		return
	}

	updateQuery := `
		update servers
		set is_docker_installation_task_running = false
		where id = :serverID
	`

	params := gin.H{
		"serverID": serverID,
	}

	if _, err := w.DB.NamedExecContext(w.Ctx, updateQuery, params); err != nil {
		w.appendLogToBuffer(serverID, models.DockerInstallationLog{
			Type:    models.LogTypeError,
			Content: shared.ErrInternalServer,
		})
		return
	}

	isLast := true
	w.appendLogToBuffer(serverID, models.DockerInstallationLog{
		Type:    models.LogTypeSystem,
		Content: "Bye!",
		IsLast:  &isLast,
	})
}

func (w *dockerInstallationWorker) appendLogToBuffer(serverID string, log models.DockerInstallationLog) {
	w.LogMu.Lock()
	defer w.LogMu.Unlock()

	if _, exists := w.LogBuf[serverID]; !exists {
		w.LogBuf[serverID] = models.DockerInstallationLogs{}
	}

	w.LogBuf[serverID] = append(w.LogBuf[serverID], log)
}

func (w *dockerInstallationWorker) flushLogs() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.LogMu.Lock()

			for serverID, logs := range w.LogBuf {
				if len(logs) == 0 {
					continue
				}

				query := `
					UPDATE servers
					SET docker_installation_logs = docker_installation_logs || :logs
					WHERE id = :serverID
				`

				params := gin.H{
					"logs":     models.DockerInstallationLogs(logs),
					"serverID": serverID,
				}

				if _, err := w.DB.NamedExecContext(w.Ctx, query, params); err != nil {
					log.Println("flushLogs() error", err)
					continue
				}

				delete(w.LogBuf, serverID)
			}

			w.LogMu.Unlock()
		}
	}
}
