package repositories

import (
	"database/sql"

	"github.com/mohit4bug/mo-sh/internal/models"
)

type UserRepository interface {
	Create(user *models.User) error
	FindByEmail(email string) (*models.User, error)
	EmailExists(email string) (bool, error)
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *models.User) error {
	_, err := r.db.Exec(
		"INSERT INTO users (email, password_hash) VALUES ($1, $2)",
		user.Email,
		user.PasswordHash,
	)
	return err
}

func (r *userRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(
		"SELECT id, password_hash FROM users WHERE email = $1",
		email,
	).Scan(&user.ID, &user.PasswordHash)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) EmailExists(email string) (bool, error) {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM users WHERE email = $1",
		email,
	).Scan(&count)
	return count > 0, err
}
