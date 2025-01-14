package repositories

import (
	"EchoAuth/database"
	"EchoAuth/models"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("record not found")
)

type UserRepository interface {
	Create(user *models.User) error
	FindByEmail(email string) (*models.User, error)
	FindByID(id uint) (*models.User, error)
	Update(user *models.User) error
	Delete(id uint) error
	FindByResetToken(token string) (*models.User, error)
}

type userRepository struct {
	db *database.DB
}

func NewUserRepository(db *database.DB) UserRepository {
	return &userRepository{
		db: db,
	}
}

func (r *userRepository) Create(user *models.User) error {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	query := `
		INSERT INTO users (email, password, first_name, last_name, password_reset_token, 
			reset_token_expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	err := r.db.QueryRow(query,
		user.Email, user.Password, user.FirstName, user.LastName,
		user.PasswordResetToken, user.ResetTokenExpiresAt,
		user.CreatedAt, user.UpdatedAt).Scan(&user.ID)

	return err
}

func (r *userRepository) FindByEmail(email string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, email, password, first_name, last_name, password_reset_token,
			reset_token_expires_at, created_at, updated_at, deleted_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL`

	err := r.db.QueryRow(query, email).Scan(
		&user.ID, &user.Email, &user.Password, &user.FirstName, &user.LastName,
		&user.PasswordResetToken, &user.ResetTokenExpiresAt,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepository) FindByID(id uint) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, email, password, first_name, last_name, password_reset_token,
			reset_token_expires_at, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL`

	err := r.db.QueryRow(query, id).Scan(
		&user.ID, &user.Email, &user.Password, &user.FirstName, &user.LastName,
		&user.PasswordResetToken, &user.ResetTokenExpiresAt,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepository) Update(user *models.User) error {
	user.UpdatedAt = time.Now()

	query := `
		UPDATE users
		SET email = $1, password = $2, first_name = $3, last_name = $4,
			password_reset_token = $5, reset_token_expires_at = $6, updated_at = $7
		WHERE id = $8 AND deleted_at IS NULL`

	result, err := r.db.Exec(query,
		user.Email, user.Password, user.FirstName, user.LastName,
		user.PasswordResetToken, user.ResetTokenExpiresAt,
		user.UpdatedAt, user.ID)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *userRepository) Delete(id uint) error {
	now := time.Now()
	query := `
		UPDATE users
		SET deleted_at = $1
		WHERE id = $2 AND deleted_at IS NULL`

	result, err := r.db.Exec(query, now, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *userRepository) FindByResetToken(token string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, email, password, first_name, last_name, password_reset_token,
			reset_token_expires_at, created_at, updated_at, deleted_at
		FROM users
		WHERE password_reset_token = $1 AND deleted_at IS NULL`

	err := r.db.QueryRow(query, token).Scan(
		&user.ID, &user.Email, &user.Password, &user.FirstName, &user.LastName,
		&user.PasswordResetToken, &user.ResetTokenExpiresAt,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}
