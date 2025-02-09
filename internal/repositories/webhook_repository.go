package repositories

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/mohit4bug/mo-sh/internal/models"
)

type WebhookRepository interface {
	SaveGithubApp(githubAppResponse *models.GithubAppResponse, sourceID string) error
}

type webhookRepository struct {
	db *sql.DB
}

func NewWebhookRepository(db *sql.DB) WebhookRepository {
	return &webhookRepository{db: db}
}

func (r *webhookRepository) SaveGithubApp(githubAppResponse *models.GithubAppResponse, sourceID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	githubApp := githubAppResponse.ToGithubApp()
	githubApp.SourceID = sourceID

	ownerJSON, err := json.Marshal(githubApp.Owner)
	if err != nil {
		return err
	}

	permissionsJSON, err := json.Marshal(githubApp.Permissions)
	if err != nil {
		return err
	}

	eventsJSON, err := json.Marshal(githubApp.Events)
	if err != nil {
		return err
	}

	var keyId string
	err = tx.QueryRow(
		`INSERT INTO keys (name, key, type, is_external) VALUES ($1, $2, $3, $4) RETURNING id`,
		fmt.Sprintf("gh-%s", githubApp.Name),
		githubAppResponse.PEM,
		"rsa",
		true,
	).Scan(&keyId)
	if err != nil {
		return err
	}

	githubApp.KeyID = keyId

	_, err = tx.Exec(
		`INSERT INTO github_apps (
			slug, client_id, node_id, owner, name, description, external_url, 
			html_url, created_at, updated_at, permissions, events, source_id, 
			client_secret, webhook_secret, key_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)`,
		githubApp.Slug,
		githubApp.ClientID,
		githubApp.NodeID,
		ownerJSON,
		githubApp.Name,
		githubApp.Description,
		githubApp.ExternalURL,
		githubApp.HTMLURL,
		githubApp.CreatedAt,
		githubApp.UpdatedAt,
		permissionsJSON,
		eventsJSON,
		sourceID,
		githubApp.ClientSecret,
		githubApp.WebhookSecret,
		keyId,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}
