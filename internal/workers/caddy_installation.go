package workers

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mohit4bug/mo-sh/internal/models"
	"github.com/mohit4bug/mo-sh/internal/shared"
	"github.com/mohit4bug/mo-sh/pkg/ssh"
	"github.com/redis/go-redis/v9"
)

const (
	CaddyPendingQueue    = "caddy_installation:pending"
	CaddyProcessingQueue = "caddy_installation:processing"
)

type caddyInstallationWorker struct {
	DB          *sqlx.DB
	RedisClient *redis.Client
	Ctx         context.Context
	LogBuf      map[string]models.CaddyInstallationLogs
	LogMu       sync.Mutex
}

func NewCaddyInstallationWorker(db *sqlx.DB, redisClient *redis.Client, ctx context.Context) *caddyInstallationWorker {
	return &caddyInstallationWorker{
		DB:          db,
		RedisClient: redisClient,
		Ctx:         ctx,
		LogBuf:      make(map[string]models.CaddyInstallationLogs),
		LogMu:       sync.Mutex{},
	}
}

func (w *caddyInstallationWorker) Start(numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		go w.worker()
	}

	go w.flushLogs()
}

func (w *caddyInstallationWorker) worker() {
	for {
		serverID, err := w.RedisClient.BRPopLPush(w.Ctx, CaddyPendingQueue, CaddyProcessingQueue, 0).Result()
		if err != nil {
			continue
		}

		w.installCaddy(serverID)

		_, err = w.RedisClient.LRem(w.Ctx, CaddyProcessingQueue, 1, serverID).Result()
		if err != nil {
			continue
		}
	}
}

func (w *caddyInstallationWorker) installCaddy(serverID string) {
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
			w.appendLogToBuffer(serverID, models.CaddyInstallationLog{
				Type:    models.LogTypeError,
				Content: shared.ErrNotFound,
			})
		} else {
			w.appendLogToBuffer(serverID, models.CaddyInstallationLog{
				Type:    models.LogTypeError,
				Content: shared.ErrInternalServer,
			})
		}
		w.finishCaddyInstall(serverID, false)
		return
	}

	sshClient := ssh.NewClient(server.Hostname, server.Port, "root", []byte(server.Key))
	if err := sshClient.Connect(); err != nil {
		w.appendLogToBuffer(serverID, models.CaddyInstallationLog{
			Type:    models.LogTypeError,
			Content: shared.ErrSSHConnection,
		})
		w.finishCaddyInstall(serverID, false)
		return
	}
	defer sshClient.Close()

	cmd := `
		sudo apt-get update
		sudo apt-get install -y debian-keyring debian-archive-keyring apt-transport-https
		curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
		curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
		sudo apt-get update
		sudo apt-get install -y caddy
	`

	err := sshClient.ExecuteWithStreams(cmd,
		func(text string) {
			w.appendLogToBuffer(serverID, models.CaddyInstallationLog{
				Type:    models.LogTypeInfo,
				Content: text,
			})
		},
		func(text string) {
			w.appendLogToBuffer(serverID, models.CaddyInstallationLog{
				Type:    models.LogTypeError,
				Content: text,
			})
		},
	)

	if err != nil {
		w.appendLogToBuffer(serverID, models.CaddyInstallationLog{
			Type:    models.LogTypeError,
			Content: shared.ErrInternalServer,
		})
		w.finishCaddyInstall(serverID, false)
		return
	}

	if !w.finishCaddyInstall(serverID, true) {
		return
	}

	isLast := true
	w.appendLogToBuffer(serverID, models.CaddyInstallationLog{
		Type:    models.LogTypeSystem,
		Content: "Bye!",
		IsLast:  &isLast,
	})
}

func (w *caddyInstallationWorker) finishCaddyInstall(serverID string, hasCaddy bool) bool {
	updateQuery := `
		update servers
		set
			is_caddy_installation_task_running = false,
			has_caddy = :hasCaddy
		where id = :serverID
	`

	params := map[string]any{
		"hasCaddy": hasCaddy,
		"serverID": serverID,
	}

	if _, err := w.DB.NamedExecContext(w.Ctx, updateQuery, params); err != nil {
		w.appendLogToBuffer(serverID, models.CaddyInstallationLog{
			Type:    models.LogTypeError,
			Content: shared.ErrInternalServer,
		})
		return false
	}

	return true
}

func (w *caddyInstallationWorker) appendLogToBuffer(serverID string, log models.CaddyInstallationLog) {
	w.LogMu.Lock()
	defer w.LogMu.Unlock()

	if _, exists := w.LogBuf[serverID]; !exists {
		w.LogBuf[serverID] = models.CaddyInstallationLogs{}
	}

	w.LogBuf[serverID] = append(w.LogBuf[serverID], log)
}

func (w *caddyInstallationWorker) flushLogs() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.LogMu.Lock()
			pendingLogs := w.LogBuf
			w.LogBuf = make(map[string]models.CaddyInstallationLogs)
			w.LogMu.Unlock()

			for serverID, logs := range pendingLogs {
				if len(logs) == 0 {
					continue
				}

				query := `
					UPDATE servers
					SET caddy_installation_logs = caddy_installation_logs || :logs
					WHERE id = :serverID
				`

				params := map[string]any{
					"logs":     models.CaddyInstallationLogs(logs),
					"serverID": serverID,
				}

				if _, err := w.DB.NamedExecContext(w.Ctx, query, params); err != nil {
					log.Println("flushLogs() error", err)
					w.LogMu.Lock()
					w.LogBuf[serverID] = append(logs, w.LogBuf[serverID]...)
					w.LogMu.Unlock()
					continue
				}
			}
		}
	}
}
