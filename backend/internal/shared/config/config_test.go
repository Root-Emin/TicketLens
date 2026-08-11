package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Defaults(t *testing.T) {
	cfg := Load()

	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "masterfabric", cfg.Database.User)
	assert.Equal(t, "localhost", cfg.Redis.Host)
	assert.Equal(t, 6379, cfg.Redis.Port)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)
}

func TestLoad_EnvironmentOverrides(t *testing.T) {
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("DB_HOST", "db.example.com")

	cfg := Load()
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "db.example.com", cfg.Database.Host)
}

func TestLoad_DBPoolInt32Bounds(t *testing.T) {
	t.Setenv("DB_MAX_CONNS", "50")
	t.Setenv("DB_MIN_CONNS", "2147483648")

	cfg := Load()
	assert.Equal(t, int32(50), cfg.Database.MaxConns)
	assert.Equal(t, int32(5), cfg.Database.MinConns)
}

func TestDatabaseConfig_DSN(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "user",
		Password: "pass",
		DBName:   "testdb",
		SSLMode:  "disable",
	}
	expected := "postgres://user:pass@localhost:5432/testdb?sslmode=disable"
	assert.Equal(t, expected, cfg.DSN())
}

func TestDatabaseConfig_DSN_EscapesSpecialCharacters(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "user@domain",
		Password: "p@ss:w?rd#",
		DBName:   "testdb",
		SSLMode:  "require",
	}
	dsn := cfg.DSN()
	assert.Contains(t, dsn, "postgres://")
	assert.Contains(t, dsn, "sslmode=require")
	assert.NotContains(t, dsn, "p@ss:w?rd#")
}

func TestRedisConfig_Addr(t *testing.T) {
	cfg := RedisConfig{Host: "redis.local", Port: 6380}
	assert.Equal(t, "redis.local:6380", cfg.Addr())
}

// A deployment that boots with the shipped defaults would sign tokens anybody
// can forge, so Validate is what turns that into a failed start instead.

func productionConfig() *Config {
	return &Config{
		Env:      EnvProduction,
		Server:   ServerConfig{CORSAllowedOrigins: []string{"https://app.example.com"}},
		Database: DatabaseConfig{Password: "a-real-database-password"},
		JWT:      JWTConfig{Secret: "0123456789abcdef0123456789abcdef"},
	}
}

func TestValidate_DevelopmentToleratesDefaults(t *testing.T) {
	// ./start.sh has no environment file; local development must keep working.
	t.Setenv("APP_ENV", "")
	cfg := Load()

	assert.Equal(t, EnvDevelopment, cfg.Env)
	assert.True(t, cfg.IsDevelopment())
	assert.NoError(t, cfg.Validate())
}

func TestValidate_ProductionAcceptsRealSecrets(t *testing.T) {
	assert.NoError(t, productionConfig().Validate())
}

func TestValidate_ProductionRejectsUnsafeDefaults(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{"default jwt secret", func(c *Config) { c.JWT.Secret = InsecureJWTSecret }},
		{"short jwt secret", func(c *Config) { c.JWT.Secret = "too-short" }},
		{"default database password", func(c *Config) { c.Database.Password = defaultDBPassword }},
		{"wildcard cors", func(c *Config) { c.Server.CORSAllowedOrigins = nil }},
		{"unknown environment", func(c *Config) { c.Env = "prod" }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := productionConfig()
			tc.mutate(cfg)
			assert.Error(t, cfg.Validate())
		})
	}
}

func TestValidate_StagingIsHeldToTheSameStandard(t *testing.T) {
	cfg := productionConfig()
	cfg.Env = EnvStaging
	assert.NoError(t, cfg.Validate())

	cfg.JWT.Secret = InsecureJWTSecret
	assert.Error(t, cfg.Validate())
}
