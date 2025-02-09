package repositories

import (
	"database/sql"

	"github.com/mohit4bug/mo-sh/internal/models"
)

type SourceRepository interface {
	Create(source *models.Source) (string, error)
	FindAll() ([]models.Source, error)
	FindByID(id string) (*models.Source, error)
}

type sourceRepository struct {
	db *sql.DB
}

func NewSourceRepository(db *sql.DB) SourceRepository {
	return &sourceRepository{db: db}
}

func (r *sourceRepository) Create(source *models.Source) (string, error) {
	var id string
	err := r.db.QueryRow(
		"INSERT INTO sources (name, type) VALUES ($1, $2) RETURNING id",
		source.Name,
		source.Type,
	).Scan(&id)
	return id, err
}

func (r *sourceRepository) FindAll() ([]models.Source, error) {
	rows, err := r.db.Query("SELECT id, name, type FROM sources")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sources := make([]models.Source, 0)
	for rows.Next() {
		var source models.Source
		if err := rows.Scan(&source.ID, &source.Name, &source.Type); err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, nil
}

func (r *sourceRepository) FindByID(id string) (*models.Source, error) {
	var source models.Source
	err := r.db.QueryRow(
		"SELECT name FROM sources WHERE id = $1",
		id,
	).Scan(&source.Name)
	if err != nil {
		return nil, err
	}
	return &source, nil
}
