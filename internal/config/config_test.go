package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Run("loads configuration with default values", func(t *testing.T) {
		cfg, err := Load()
		require.NoError(t, err)
		require.NotNil(t, cfg)

		assert.Equal(t, 9090, cfg.Server.Port)
		assert.Equal(t, "0.0.0.0", cfg.Server.Host)
		assert.Equal(t, "localhost", cfg.Database.Host)
		assert.Equal(t, 5432, cfg.Database.Port)
		assert.Equal(t, "postgres", cfg.Database.User)
		assert.Equal(t, "ledger", cfg.Database.DBName)
		assert.Equal(t, "disable", cfg.Database.SSLMode)
	})

	t.Run("loads configuration from environment variables", func(t *testing.T) {
		os.Setenv("SERVER_PORT", "8080")
		os.Setenv("DB_HOST", "testhost")
		os.Setenv("DB_PORT", "5433")
		os.Setenv("DB_NAME", "testdb")
		defer func() {
			os.Unsetenv("SERVER_PORT")
			os.Unsetenv("DB_HOST")
			os.Unsetenv("DB_PORT")
			os.Unsetenv("DB_NAME")
		}()

		cfg, err := Load()
		require.NoError(t, err)

		assert.Equal(t, 8080, cfg.Server.Port)
		assert.Equal(t, "testhost", cfg.Database.Host)
		assert.Equal(t, 5433, cfg.Database.Port)
		assert.Equal(t, "testdb", cfg.Database.DBName)
	})
}

func TestDatabaseConfig_ConnectionString(t *testing.T) {
	cfg := &DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "testuser",
		Password: "testpass",
		DBName:   "testdb",
		SSLMode:  "disable",
	}

	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"
	assert.Equal(t, expected, cfg.ConnectionString())
}

func TestGetEnv(t *testing.T) {
	t.Run("returns environment variable value", func(t *testing.T) {
		os.Setenv("TEST_VAR", "test_value")
		defer os.Unsetenv("TEST_VAR")

		value := getEnv("TEST_VAR", "default")
		assert.Equal(t, "test_value", value)
	})

	t.Run("returns default value when environment variable is not set", func(t *testing.T) {
		value := getEnv("NON_EXISTENT_VAR", "default")
		assert.Equal(t, "default", value)
	})
}

func TestGetEnvAsInt(t *testing.T) {
	t.Run("returns integer from environment variable", func(t *testing.T) {
		os.Setenv("TEST_INT", "42")
		defer os.Unsetenv("TEST_INT")

		value := getEnvAsInt("TEST_INT", 10)
		assert.Equal(t, 42, value)
	})

	t.Run("returns default value when environment variable is not set", func(t *testing.T) {
		value := getEnvAsInt("NON_EXISTENT_INT", 10)
		assert.Equal(t, 10, value)
	})

	t.Run("returns default value when environment variable is not a valid integer", func(t *testing.T) {
		os.Setenv("TEST_INVALID_INT", "not_a_number")
		defer os.Unsetenv("TEST_INVALID_INT")

		value := getEnvAsInt("TEST_INVALID_INT", 10)
		assert.Equal(t, 10, value)
	})
}
