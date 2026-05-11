package workers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mohit4bug/mo-sh/internal/models"
	"github.com/mohit4bug/mo-sh/internal/shared"
	"github.com/mohit4bug/mo-sh/pkg/ssh"
	"github.com/redis/go-redis/v9"
)

const (
	DatabaseDeploymentPendingQueue    = "database_deployment:pending"
	DatabaseDeploymentProcessingQueue = "database_deployment:processing"
)

type databaseDeploymentWorker struct {
	DB          *sqlx.DB
	RedisClient *redis.Client
	Ctx         context.Context
	LogBuf      map[string]models.DatabaseDeploymentLogs
	LogMu       sync.Mutex
}

type databaseDeployment struct {
	models.DatabaseService
	JobID          string `db:"job_id"`
	ServerHostname string `db:"server_hostname"`
	ServerPort     int    `db:"server_port"`
	SSHKey         string `db:"ssh_key"`
	PlanImage      string `db:"plan_image"`
}

func NewDatabaseDeploymentWorker(db *sqlx.DB, redisClient *redis.Client, ctx context.Context) *databaseDeploymentWorker {
	return &databaseDeploymentWorker{
		DB:          db,
		RedisClient: redisClient,
		Ctx:         ctx,
		LogBuf:      make(map[string]models.DatabaseDeploymentLogs),
		LogMu:       sync.Mutex{},
	}
}

func (w *databaseDeploymentWorker) Start(numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		go w.worker()
	}

	go w.flushLogs()
}

func (w *databaseDeploymentWorker) worker() {
	for {
		jobID, err := w.RedisClient.BRPopLPush(w.Ctx, DatabaseDeploymentPendingQueue, DatabaseDeploymentProcessingQueue, 0).Result()
		if err != nil {
			continue
		}

		w.deployDatabase(jobID)

		_, err = w.RedisClient.LRem(w.Ctx, DatabaseDeploymentProcessingQueue, 1, jobID).Result()
		if err != nil {
			continue
		}
	}
}

func (w *databaseDeploymentWorker) deployDatabase(jobID string) {
	var deployment databaseDeployment
	query := `
		select
			j.id as job_id,
			ds.id,
			ds.user_id,
			ds.server_id,
			ds.plan_id,
			ds.name,
			ds.database_name,
			ds.database_user,
			ds.database_password,
			ds.container_name,
			ds.volume_name,
			ds.network_name,
			ds.internal_port,
			ds.public_port,
			ds.expose_publicly,
			ds.status,
			ds.deployment_logs,
			ds.created_at,
			ds.updated_at,
			s.hostname as server_hostname,
			s.port as server_port,
			k.key as ssh_key,
			dp.image as plan_image
		from database_deployment_jobs j
		inner join database_services ds on ds.id = j.database_service_id
		inner join database_plans dp on dp.id = ds.plan_id
		inner join servers s on s.id = ds.server_id
		inner join keys k on k.id = s.key_id
		where j.id = $1
	`
	if err := w.DB.GetContext(w.Ctx, &deployment, query, jobID); err != nil {
		log.Println("database deployment lookup error", err)
		return
	}

	serviceID := deployment.ID
	if !w.markDatabaseDeploymentStarted(jobID, serviceID) {
		return
	}

	w.appendLogToBuffer(serviceID, models.DatabaseDeploymentLog{
		Type:    models.LogTypeSystem,
		Content: "Starting database deployment.",
	})

	sshClient := ssh.NewClient(deployment.ServerHostname, deployment.ServerPort, "root", []byte(deployment.SSHKey))
	if err := sshClient.Connect(); err != nil {
		w.failDatabaseDeployment(jobID, serviceID, shared.ErrSSHConnection)
		return
	}
	defer sshClient.Close()

	if !sshClient.CheckCommand("docker --version") {
		w.failDatabaseDeployment(jobID, serviceID, "Docker is not installed on the target server.")
		return
	}
	if !sshClient.CheckCommand("docker compose version") {
		w.failDatabaseDeployment(jobID, serviceID, "Docker Compose is not installed on the target server.")
		return
	}

	remoteDir := fmt.Sprintf("/opt/mo-sh/databases/%s", deployment.ID)
	if err := w.runRemoteCommand(sshClient, serviceID, buildDeploymentCommand(remoteDir, deployment)); err != nil {
		w.failDatabaseDeployment(jobID, serviceID, shared.ErrInternalServer)
		return
	}

	if !w.finishDatabaseDeployment(jobID, serviceID) {
		return
	}

	isLast := true
	w.appendLogToBuffer(serviceID, models.DatabaseDeploymentLog{
		Type:    models.LogTypeSystem,
		Content: "Database deployment finished.",
		IsLast:  &isLast,
	})
}

func (w *databaseDeploymentWorker) runRemoteCommand(sshClient *ssh.Client, serviceID string, cmd string) error {
	return sshClient.ExecuteWithStreams(cmd,
		func(text string) {
			w.appendLogToBuffer(serviceID, models.DatabaseDeploymentLog{
				Type:    models.LogTypeInfo,
				Content: text,
			})
		},
		func(text string) {
			w.appendLogToBuffer(serviceID, models.DatabaseDeploymentLog{
				Type:    models.LogTypeError,
				Content: text,
			})
		},
	)
}

func (w *databaseDeploymentWorker) markDatabaseDeploymentStarted(jobID string, serviceID string) bool {
	tx, err := w.DB.BeginTxx(w.Ctx, nil)
	if err != nil {
		w.appendLogToBuffer(serviceID, models.DatabaseDeploymentLog{Type: models.LogTypeError, Content: shared.ErrInternalServer})
		return false
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(w.Ctx, `update database_deployment_jobs set status = 'deploying', started_at = now() where id = $1`, jobID); err != nil {
		w.appendLogToBuffer(serviceID, models.DatabaseDeploymentLog{Type: models.LogTypeError, Content: shared.ErrInternalServer})
		return false
	}
	if _, err := tx.ExecContext(w.Ctx, `update database_services set status = 'deploying', updated_at = now() where id = $1`, serviceID); err != nil {
		w.appendLogToBuffer(serviceID, models.DatabaseDeploymentLog{Type: models.LogTypeError, Content: shared.ErrInternalServer})
		return false
	}
	if err := tx.Commit(); err != nil {
		w.appendLogToBuffer(serviceID, models.DatabaseDeploymentLog{Type: models.LogTypeError, Content: shared.ErrInternalServer})
		return false
	}
	return true
}

func (w *databaseDeploymentWorker) finishDatabaseDeployment(jobID string, serviceID string) bool {
	tx, err := w.DB.BeginTxx(w.Ctx, nil)
	if err != nil {
		w.appendLogToBuffer(serviceID, models.DatabaseDeploymentLog{Type: models.LogTypeError, Content: shared.ErrInternalServer})
		return false
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(w.Ctx, `update database_deployment_jobs set status = 'running', finished_at = now() where id = $1`, jobID); err != nil {
		w.appendLogToBuffer(serviceID, models.DatabaseDeploymentLog{Type: models.LogTypeError, Content: shared.ErrInternalServer})
		return false
	}
	if _, err := tx.ExecContext(w.Ctx, `update database_services set status = 'running', updated_at = now() where id = $1`, serviceID); err != nil {
		w.appendLogToBuffer(serviceID, models.DatabaseDeploymentLog{Type: models.LogTypeError, Content: shared.ErrInternalServer})
		return false
	}
	if err := tx.Commit(); err != nil {
		w.appendLogToBuffer(serviceID, models.DatabaseDeploymentLog{Type: models.LogTypeError, Content: shared.ErrInternalServer})
		return false
	}
	return true
}

func (w *databaseDeploymentWorker) failDatabaseDeployment(jobID string, serviceID string, message string) {
	w.appendLogToBuffer(serviceID, models.DatabaseDeploymentLog{
		Type:    models.LogTypeError,
		Content: message,
	})

	_, _ = w.DB.ExecContext(
		w.Ctx,
		`update database_deployment_jobs set status = 'failed', error_message = $1, finished_at = now() where id = $2`,
		message,
		jobID,
	)
	_, _ = w.DB.ExecContext(
		w.Ctx,
		`update database_services set status = 'failed', updated_at = now() where id = $1`,
		serviceID,
	)

	isLast := true
	w.appendLogToBuffer(serviceID, models.DatabaseDeploymentLog{
		Type:    models.LogTypeSystem,
		Content: "Database deployment failed.",
		IsLast:  &isLast,
	})
}

func (w *databaseDeploymentWorker) appendLogToBuffer(serviceID string, log models.DatabaseDeploymentLog) {
	w.LogMu.Lock()
	defer w.LogMu.Unlock()

	if _, exists := w.LogBuf[serviceID]; !exists {
		w.LogBuf[serviceID] = models.DatabaseDeploymentLogs{}
	}

	w.LogBuf[serviceID] = append(w.LogBuf[serviceID], log)
}

func (w *databaseDeploymentWorker) flushLogs() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.LogMu.Lock()
			pendingLogs := w.LogBuf
			w.LogBuf = make(map[string]models.DatabaseDeploymentLogs)
			w.LogMu.Unlock()

			for serviceID, logs := range pendingLogs {
				if len(logs) == 0 {
					continue
				}

				query := `
					update database_services
					set deployment_logs = deployment_logs || :logs
					where id = :serviceID
				`

				params := map[string]any{
					"logs":      models.DatabaseDeploymentLogs(logs),
					"serviceID": serviceID,
				}

				if _, err := w.DB.NamedExecContext(w.Ctx, query, params); err != nil {
					log.Println("database flushLogs() error", err)
					w.LogMu.Lock()
					w.LogBuf[serviceID] = append(logs, w.LogBuf[serviceID]...)
					w.LogMu.Unlock()
					continue
				}
			}
		}
	}
}

func buildDeploymentCommand(remoteDir string, deployment databaseDeployment) string {
	compose := buildComposeFile(deployment)
	env := buildDatabaseEnvFile(deployment.DatabaseService)

	return fmt.Sprintf(`
set -e
mkdir -p %[1]s
docker network create %[2]s >/dev/null 2>&1 || true
cat > %[1]s/docker-compose.yml <<'MO_SH_COMPOSE'
%[3]s
MO_SH_COMPOSE
cat > %[1]s/.env <<'MO_SH_ENV'
%[4]s
MO_SH_ENV
chmod 600 %[1]s/.env
cd %[1]s
docker compose up -d
docker inspect %[5]s >/dev/null
for attempt in $(seq 1 30); do
	if docker exec %[5]s pg_isready -U %[6]s -d %[7]s >/dev/null 2>&1; then
		break
	fi
	if [ "$attempt" -eq 30 ]; then
		docker logs --tail=100 %[5]s
		exit 1
	fi
	sleep 1
done
docker logs --tail=100 %[5]s
`,
		shellQuote(remoteDir),
		shellQuote(deployment.NetworkName),
		compose,
		env,
		shellQuote(deployment.ContainerName),
		shellQuote(deployment.DatabaseUser),
		shellQuote(deployment.DatabaseName),
	)
}

func buildComposeFile(deployment databaseDeployment) string {
	var portsBlock string
	if deployment.ExposePublicly && deployment.PublicPort.Valid {
		portsBlock = fmt.Sprintf(`
    ports:
      - "%d:%d"
`, deployment.PublicPort.Int64, deployment.InternalPort)
	}

	return fmt.Sprintf(`services:
  postgres:
    image: %s
    container_name: %s
    restart: unless-stopped
    env_file:
      - .env
    volumes:
      - %s:/var/lib/postgresql
    networks:
      - %s
%s
volumes:
  %s:

networks:
  %s:
    external: true
`,
		deployment.PlanImage,
		deployment.ContainerName,
		deployment.VolumeName,
		deployment.NetworkName,
		portsBlock,
		deployment.VolumeName,
		deployment.NetworkName,
	)
}

func buildDatabaseEnvFile(service models.DatabaseService) string {
	return fmt.Sprintf(`POSTGRES_DB=%s
POSTGRES_USER=%s
POSTGRES_PASSWORD=%s
`,
		service.DatabaseName,
		service.DatabaseUser,
		service.DatabasePassword,
	)
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
