package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/mohit4bug/mo-sh/internal/models"
	"github.com/mohit4bug/mo-sh/internal/shared"
	"github.com/mohit4bug/mo-sh/internal/workers"
	"github.com/mohit4bug/mo-sh/pkg/session"
	"github.com/redis/go-redis/v9"
)

type DatabaseRepository interface {
	Create(c *gin.Context)
	FindByServer(c *gin.Context)
	FindByID(c *gin.Context)
	GetPendingLogs(c *gin.Context)
}

type databaseRepository struct {
	DB          *sqlx.DB
	RedisClient *redis.Client
	Ctx         context.Context
}

func NewDatabaseRepository(db *sqlx.DB, redisClient *redis.Client, ctx context.Context) *databaseRepository {
	return &databaseRepository{
		DB:          db,
		RedisClient: redisClient,
		Ctx:         ctx,
	}
}

var (
	databaseIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,62}$`)
	databaseServicePattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,62}$`)
)

func (r *databaseRepository) Create(c *gin.Context) {
	serverID := c.Param("serverID")
	userSession, ok := currentSession(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input models.CreateDatabaseService
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateCreateDatabaseService(input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var server models.Server
	if err := r.DB.GetContext(r.Ctx, &server, "select * from servers where id = $1", serverID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": shared.ErrNotFound})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}
	if !server.HasDocker {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Docker must be installed on this server before deploying databases."})
		return
	}

	var plan models.DatabasePlan
	if err := r.DB.GetContext(
		r.Ctx,
		&plan,
		"select * from database_plans where engine = $1 and version = $2 and is_active = true",
		input.Engine,
		input.Version,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported database engine or version."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}

	password := shared.GenerateRandomString(48)
	if password == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}

	suffix := shared.GenerateRandomString(12)
	if suffix == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}

	containerName := fmt.Sprintf("mo-sh-%s-%s", input.Engine, suffix)
	volumeName := fmt.Sprintf("%s-data", containerName)
	publicPort := sql.NullInt64{}
	if input.ExposePublicly {
		publicPort = sql.NullInt64{Int64: int64(*input.PublicPort), Valid: true}
	}

	tx, err := r.DB.BeginTxx(r.Ctx, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var service models.DatabaseService
	serviceQuery := `
		insert into database_services (
			user_id,
			server_id,
			plan_id,
			name,
			database_name,
			database_user,
			database_password,
			container_name,
			volume_name,
			network_name,
			internal_port,
			public_port,
			expose_publicly
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'mo-sh', $10, $11, $12)
		returning *
	`
	if err := tx.GetContext(
		r.Ctx,
		&service,
		serviceQuery,
		userSession.UserID,
		serverID,
		plan.ID,
		input.Name,
		input.DatabaseName,
		input.DatabaseUser,
		password,
		containerName,
		volumeName,
		plan.DefaultPort,
		publicPort,
		input.ExposePublicly,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}

	var job models.DatabaseDeploymentJob
	if err := tx.GetContext(
		r.Ctx,
		&job,
		"insert into database_deployment_jobs (database_service_id) values ($1) returning *",
		service.ID,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}

	if err := r.RedisClient.LPush(r.Ctx, workers.DatabaseDeploymentPendingQueue, job.ID).Err(); err != nil {
		_, _ = r.DB.ExecContext(
			r.Ctx,
			`update database_deployment_jobs set status = 'failed', error_message = $1, finished_at = now() where id = $2`,
			"Failed to queue deployment job.",
			job.ID,
		)
		_, _ = r.DB.ExecContext(
			r.Ctx,
			`update database_services set status = 'failed', updated_at = now() where id = $1`,
			service.ID,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Database deployment queued.",
		"data": gin.H{
			"database": models.NewDatabaseServiceResponse(service, server.Hostname),
			"job":      job,
		},
	})
}

func (r *databaseRepository) FindByServer(c *gin.Context) {
	serverID := c.Param("serverID")

	var server models.Server
	if err := r.DB.GetContext(r.Ctx, &server, "select * from servers where id = $1", serverID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": shared.ErrNotFound})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}

	services := []models.DatabaseService{}
	if err := r.DB.SelectContext(r.Ctx, &services, "select * from database_services where server_id = $1 order by created_at desc", serverID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}

	databases := make([]models.DatabaseServiceResponse, 0, len(services))
	for _, service := range services {
		databases = append(databases, models.NewDatabaseServiceResponse(service, server.Hostname))
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
		"data":    gin.H{"databases": databases},
	})
}

func (r *databaseRepository) FindByID(c *gin.Context) {
	databaseID := c.Param("databaseID")

	type DatabaseServiceWithServer struct {
		models.DatabaseService
		ServerHostname string `db:"server_hostname"`
	}

	var service DatabaseServiceWithServer
	if err := r.DB.GetContext(
		r.Ctx,
		&service,
		`
			select
				ds.*,
				s.hostname as server_hostname
			from database_services ds
			inner join servers s on s.id = ds.server_id
			where ds.id = $1
		`,
		databaseID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": shared.ErrNotFound})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
		"data": gin.H{
			"database": models.NewDatabaseServiceResponse(service.DatabaseService, service.ServerHostname),
		},
	})
}

func (r *databaseRepository) GetPendingLogs(c *gin.Context) {
	databaseID := c.Param("databaseID")

	tx, err := r.DB.BeginTx(r.Ctx, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var exists bool
	if err = tx.QueryRowContext(r.Ctx, `select exists(select 1 from database_services where id = $1)`, databaseID).Scan(&exists); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": shared.ErrNotFound})
		return
	}

	var logs models.DatabaseDeploymentLogs
	query := `
		select
			coalesce(jsonb_agg(logs), '[]'::jsonb) as unprocessed_logs
		from (
			select
				jsonb_array_elements(coalesce(deployment_logs, '[]'::jsonb)) as logs
			from
				database_services
			where
				id = $1
		) as logs
		where
			logs->>'processedAt' is null
	`

	if err = tx.QueryRowContext(r.Ctx, query, databaseID).Scan(&logs); err != nil {
		if err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
			return
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	updateQuery := `
		update database_services
		set deployment_logs = coalesce((
			select jsonb_agg(
				case
					when elem->>'processedAt' is null
					then jsonb_set(elem, '{processedAt}', to_jsonb($1::text))
					else elem
				end
			)
			from jsonb_array_elements(deployment_logs) as elem
		), '[]'::jsonb)
		where id = $2
	`

	if _, err = tx.ExecContext(r.Ctx, updateQuery, now, databaseID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": shared.ErrInternalServer})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
		"data":    gin.H{"logs": logs},
	})
}

func currentSession(c *gin.Context) (*session.Session, bool) {
	value, exists := c.Get("session")
	if !exists {
		return nil, false
	}

	userSession, ok := value.(*session.Session)
	return userSession, ok
}

func validateCreateDatabaseService(input models.CreateDatabaseService) error {
	if input.Engine != "postgres" {
		return fmt.Errorf("only postgres databases are currently supported")
	}
	if !databaseServicePattern.MatchString(input.Name) {
		return fmt.Errorf("name must contain only letters, numbers, and dashes")
	}
	if !databaseIdentifierPattern.MatchString(input.DatabaseName) {
		return fmt.Errorf("databaseName must be a valid PostgreSQL identifier")
	}
	if !databaseIdentifierPattern.MatchString(input.DatabaseUser) {
		return fmt.Errorf("databaseUser must be a valid PostgreSQL identifier")
	}
	if input.ExposePublicly {
		if input.PublicPort == nil {
			return fmt.Errorf("publicPort is required when exposePublicly is true")
		}
		if *input.PublicPort < 1024 || *input.PublicPort > 65535 {
			return fmt.Errorf("publicPort must be between 1024 and 65535")
		}
	}
	return nil
}
