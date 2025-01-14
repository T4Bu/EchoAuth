package repositories

import (
	"EchoAuth/models"
	"fmt"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	// Setup test database
	dsn := "host=localhost user=postgres password=postgres dbname=auth_test_db port=5433 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("Failed to connect to test database: %v\n", err)
		os.Exit(1)
	}

	// Migrate the schema
	err = db.AutoMigrate(&models.User{}, &models.RefreshToken{})
	if err != nil {
		fmt.Printf("Failed to migrate test database: %v\n", err)
		os.Exit(1)
	}

	testDB = db

	// Run tests
	code := m.Run()

	// Cleanup
	sqlDB, err := testDB.DB()
	if err == nil {
		sqlDB.Close()
	}

	os.Exit(code)
}

func setupTest() (*userRepository, func()) {
	// Clear the database before each test
	testDB.Exec("DELETE FROM users")

	repo := &userRepository{db: testDB}

	return repo, func() {
		testDB.Exec("DELETE FROM users")
	}
}

func TestCreateUser(t *testing.T) {
	repo, cleanup := setupTest()
	defer cleanup()

	user := &models.User{
		Email:     "test@example.com",
		Password:  "hashed_password",
		FirstName: "John",
		LastName:  "Doe",
	}

	err := repo.Create(user)
	if err != nil {
		t.Errorf("Failed to create user: %v", err)
	}

	if user.ID == 0 {
		t.Error("Expected user ID to be set after creation")
	}
}

func TestFindByEmail(t *testing.T) {
	repo, cleanup := setupTest()
	defer cleanup()

	// Create test user
	user := &models.User{
		Email:     "test@example.com",
		Password:  "hashed_password",
		FirstName: "John",
		LastName:  "Doe",
	}
	repo.Create(user)

	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "Existing user",
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "Non-existent user",
			email:   "nonexistent@example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := repo.FindByEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindByEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && found.Email != tt.email {
				t.Errorf("FindByEmail() got = %v, want %v", found.Email, tt.email)
			}
		})
	}
}

func TestFindByID(t *testing.T) {
	repo, cleanup := setupTest()
	defer cleanup()

	// Create test user
	user := &models.User{
		Email:     "test@example.com",
		Password:  "hashed_password",
		FirstName: "John",
		LastName:  "Doe",
	}
	repo.Create(user)

	tests := []struct {
		name    string
		id      uint
		wantErr bool
	}{
		{
			name:    "Existing user",
			id:      user.ID,
			wantErr: false,
		},
		{
			name:    "Non-existent user",
			id:      999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := repo.FindByID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && found.ID != tt.id {
				t.Errorf("FindByID() got = %v, want %v", found.ID, tt.id)
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {
	repo, cleanup := setupTest()
	defer cleanup()

	// Create test user
	user := &models.User{
		Email:     "test@example.com",
		Password:  "hashed_password",
		FirstName: "John",
		LastName:  "Doe",
	}
	repo.Create(user)

	// Update user
	user.FirstName = "Jane"
	err := repo.Update(user)
	if err != nil {
		t.Errorf("Failed to update user: %v", err)
	}

	// Verify update
	updated, err := repo.FindByID(user.ID)
	if err != nil {
		t.Errorf("Failed to find updated user: %v", err)
	}
	if updated.FirstName != "Jane" {
		t.Errorf("Update failed: expected FirstName = Jane, got %s", updated.FirstName)
	}
}

func TestDeleteUser(t *testing.T) {
	repo, cleanup := setupTest()
	defer cleanup()

	// Create test user
	user := &models.User{
		Email:     "test@example.com",
		Password:  "hashed_password",
		FirstName: "John",
		LastName:  "Doe",
	}
	repo.Create(user)

	// Delete user
	err := repo.Delete(user.ID)
	if err != nil {
		t.Errorf("Failed to delete user: %v", err)
	}

	// Verify deletion
	_, err = repo.FindByID(user.ID)
	if err == nil {
		t.Error("Expected error when finding deleted user")
	}
}
