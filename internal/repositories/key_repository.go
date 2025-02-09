package repositories

import (
	"database/sql"

	"github.com/mohit4bug/mo-sh/internal/models"
)

type KeyRepository interface {
	Create(key *models.Key) error
	FindAll() ([]models.Key, error)
	FindByID(id string) (*models.Key, error)
}

type keyRepository struct {
	db *sql.DB
}

func NewKeyRepository(db *sql.DB) KeyRepository {
	return &keyRepository{db: db}
}

func (r *keyRepository) Create(key *models.Key) error {
	_, err := r.db.Exec(
		"INSERT INTO keys (name, key, type, is_external) VALUES ($1, $2, $3, $4)",
		key.Name, key.Key, key.Type, false,
	)
	return err
}

func (r *keyRepository) FindAll() ([]models.Key, error) {
	rows, err := r.db.Query("SELECT id, name, is_external FROM keys")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := make([]models.Key, 0)
	for rows.Next() {
		var key models.Key
		if err := rows.Scan(&key.ID, &key.Name, &key.IsExternal); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func (r *keyRepository) FindByID(id string) (*models.Key, error) {
	var key models.Key
	err := r.db.QueryRow(
		"SELECT id, name, key, type FROM keys WHERE id = $1",
		id,
	).Scan(&key.ID, &key.Name, &key.Key, &key.Type)
	if err != nil {
		return nil, err
	}
	return &key, nil
}
