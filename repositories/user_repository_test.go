package repositories

import (
	"EchoAuth/database"
	"EchoAuth/models"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func setupTestDB(t *testing.T) (*database.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}

	db := &database.DB{DB: mockDB}
	return db, mock
}

func TestUserRepository_Create(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)

	user := &models.User{
		Email:     "test@example.com",
		Password:  "hashedpassword",
		FirstName: "Test",
		LastName:  "User",
	}

	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs(user.Email, user.Password, user.FirstName, user.LastName,
			user.PasswordResetToken, user.ResetTokenExpiresAt,
			sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	err := repo.Create(user)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), user.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_FindByEmail(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	email := "test@example.com"

	mock.ExpectQuery(`SELECT .+ FROM users WHERE email = \$1`).
		WithArgs(email).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "password", "first_name", "last_name",
			"password_reset_token", "reset_token_expires_at",
			"created_at", "updated_at", "deleted_at",
		}).AddRow(
			1, email, "hashedpassword", "Test", "User",
			"", time.Time{}, time.Now(), time.Now(), nil,
		))

	user, err := repo.FindByEmail(email)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, email, user.Email)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_FindByEmail_NotFound(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	email := "nonexistent@example.com"

	mock.ExpectQuery(`SELECT .+ FROM users WHERE email = \$1`).
		WithArgs(email).
		WillReturnError(sql.ErrNoRows)

	user, err := repo.FindByEmail(email)
	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
	assert.Nil(t, user)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Update(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	user := &models.User{
		ID:        1,
		Email:     "test@example.com",
		FirstName: "Updated",
		LastName:  "User",
	}

	mock.ExpectExec(`UPDATE users SET .+ WHERE id = \$8`).
		WithArgs(
			user.Email, user.Password, user.FirstName, user.LastName,
			user.PasswordResetToken, user.ResetTokenExpiresAt,
			sqlmock.AnyArg(), user.ID,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Update(user)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Delete(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	userID := uint(1)

	mock.ExpectExec(`UPDATE users SET deleted_at = \$1 WHERE id = \$2`).
		WithArgs(sqlmock.AnyArg(), userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(userID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_FindByResetToken(t *testing.T) {
	db, mock := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	token := "reset-token"

	mock.ExpectQuery(`SELECT .+ FROM users WHERE password_reset_token = \$1`).
		WithArgs(token).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "password", "first_name", "last_name",
			"password_reset_token", "reset_token_expires_at",
			"created_at", "updated_at", "deleted_at",
		}).AddRow(
			1, "test@example.com", "hashedpassword", "Test", "User",
			token, time.Now().Add(time.Hour), time.Now(), time.Now(), nil,
		))

	user, err := repo.FindByResetToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, token, user.PasswordResetToken)
	assert.NoError(t, mock.ExpectationsWereMet())
}
