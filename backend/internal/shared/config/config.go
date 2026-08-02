package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration.
type Config struct {
	Env        string
	Server     ServerConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	JWT        JWTConfig
	Kafka      KafkaConfig
	WebSocket  WebSocketConfig
	Log        LogConfig
	Classifier ClassifierConfig
}

// Recognized values for APP_ENV.
const (
	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvProduction  = "production"
)

// InsecureJWTSecret is the placeholder shipped for local development. It is
// rejected outside EnvDevelopment so a deployment can never sign tokens with a
// value that is public knowledge.
const InsecureJWTSecret = "change-me-in-production"

// minJWTSecretLength is the shortest secret accepted outside development. HS256
// keys shorter than the hash output add no strength.
const minJWTSecretLength = 32

// IsDevelopment reports whether the process is running in local development,
// the only mode where insecure defaults are tolerated.
func (c *Config) IsDevelopment() bool { return c.Env == EnvDevelopment }

// Validate rejects configurations that are unsafe to serve traffic with.
//
// Development is deliberately permissive so `./start.sh` keeps working with no
// environment file. Every other environment must supply real secrets: a
// misconfigured deployment should fail loudly at boot rather than accept
// forged tokens for the rest of its life.
func (c *Config) Validate() error {
	if c.IsDevelopment() {
		return nil
	}

	switch c.Env {
	case EnvStaging, EnvProduction:
	default:
		return fmt.Errorf("APP_ENV must be one of %q, %q, %q (got %q)",
			EnvDevelopment, EnvStaging, EnvProduction, c.Env)
	}

	if c.JWT.Secret == InsecureJWTSecret {
		return fmt.Errorf("JWT_SECRET is still the development default; set a real secret for APP_ENV=%s", c.Env)
	}
	if len(c.JWT.Secret) < minJWTSecretLength {
		return fmt.Errorf("JWT_SECRET must be at least %d characters for APP_ENV=%s (got %d)",
			minJWTSecretLength, c.Env, len(c.JWT.Secret))
	}
	if c.Database.Password == defaultDBPassword {
		return fmt.Errorf("DB_PASSWORD is still the development default; set a real password for APP_ENV=%s", c.Env)
	}
	if len(c.Server.CORSAllowedOrigins) == 0 {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS must list explicit origins for APP_ENV=%s", c.Env)
	}

	return nil
}

// WebSocketConfig holds real-time WebSocket settings.
type WebSocketConfig struct {
	Enabled         bool
	MaxConnections  int
	PingIntervalSec int
	ReadBufferSize  int
	WriteBufferSize int
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host               string
	Port               int
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	IdleTimeout        time.Duration
	CORSAllowedOrigins []string
	MaxBodyBytes       int64
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxConns int32
	MinConns int32
}

// DSN returns the PostgreSQL connection string with escaped credentials.
func (d DatabaseConfig) DSN() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(d.User, d.Password),
		Host:   fmt.Sprintf("%s:%d", d.Host, d.Port),
		Path:   "/" + d.DBName,
	}
	u.RawQuery = url.Values{"sslmode": {d.SSLMode}}.Encode()
	return u.String()
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// Addr returns the Redis address string.
func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// JWTConfig holds JWT signing settings.
type JWTConfig struct {
	Secret          string
	ExpirationHours int
	Issuer          string
}

// KafkaConfig holds Kafka connection and consumer settings.
type KafkaConfig struct {
	Brokers           []string
	GroupID           string
	Enabled           bool
	NumPartitions     int
	ReplicationFactor int
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string // debug, info, warn, error
	Format string // json, text
}

// ClassifierConfig holds ticket classification settings.
type ClassifierConfig struct {
	// ReviewThreshold is the confidence below which an analysis is flagged for
	// human review. Applied to both the priority and the category score.
	ReviewThreshold float64

	// URL is the base URL of the Python inference service (e.g. http://localhost:8091).
	// Empty means the in-process keyword stub is used.
	URL string

	// Timeout bounds a single HTTP classify call.
	Timeout time.Duration

	// MaxRetries is how many times a failed HTTP call is retried before giving
	// up (or falling back to the stub when FallbackToStub is set).
	MaxRetries int

	// FallbackToStub uses the keyword stub after HTTP retries are exhausted so
	// tickets are still analyzed when the model service is down.
	FallbackToStub bool
}

// defaultDBPassword is the local-development database password baked into
// docker-compose. Rejected outside development by Validate.
const defaultDBPassword = "masterfabric"

// Load reads configuration from environment variables with sensible defaults.
//
// Defaults target local development. Call Validate before serving traffic to
// reject those defaults in any other environment.
func Load() *Config {
	return &Config{
		Env: envOrDefault("APP_ENV", EnvDevelopment),
		Server: ServerConfig{
			Host:               envOrDefault("SERVER_HOST", "0.0.0.0"),
			Port:               envOrDefaultInt("SERVER_PORT", 8080),
			ReadTimeout:        time.Duration(envOrDefaultInt("SERVER_READ_TIMEOUT_SECONDS", 15)) * time.Second,
			WriteTimeout:       time.Duration(envOrDefaultInt("SERVER_WRITE_TIMEOUT_SECONDS", 15)) * time.Second,
			IdleTimeout:        time.Duration(envOrDefaultInt("SERVER_IDLE_TIMEOUT_SECONDS", 60)) * time.Second,
			CORSAllowedOrigins: envOrDefaultSlice("CORS_ALLOWED_ORIGINS", nil),
			MaxBodyBytes:       envOrDefaultInt64("MAX_BODY_BYTES", 1<<20),
		},
		Database: DatabaseConfig{
			Host:     envOrDefault("DB_HOST", "localhost"),
			Port:     envOrDefaultInt("DB_PORT", 5432),
			User:     envOrDefault("DB_USER", "masterfabric"),
			Password: envOrDefault("DB_PASSWORD", defaultDBPassword),
			DBName:   envOrDefault("DB_NAME", "masterfabric"),
			SSLMode:  envOrDefault("DB_SSLMODE", "disable"),
			MaxConns: envOrDefaultInt32("DB_MAX_CONNS", 25),
			MinConns: envOrDefaultInt32("DB_MIN_CONNS", 5),
		},
		Redis: RedisConfig{
			Host:     envOrDefault("REDIS_HOST", "localhost"),
			Port:     envOrDefaultInt("REDIS_PORT", 6379),
			Password: envOrDefault("REDIS_PASSWORD", ""),
			DB:       envOrDefaultInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:          envOrDefault("JWT_SECRET", InsecureJWTSecret),
			ExpirationHours: envOrDefaultInt("JWT_EXPIRATION_HOURS", 24),
			Issuer:          envOrDefault("JWT_ISSUER", "masterfabric"),
		},
		Kafka: KafkaConfig{
			Brokers:           envOrDefaultSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			GroupID:           envOrDefault("KAFKA_GROUP_ID", "masterfabric-go"),
			Enabled:           envOrDefault("KAFKA_ENABLED", "false") == "true",
			NumPartitions:     envOrDefaultInt("KAFKA_NUM_PARTITIONS", 3),
			ReplicationFactor: envOrDefaultInt("KAFKA_REPLICATION_FACTOR", 1),
		},
		WebSocket: WebSocketConfig{
			Enabled:         envOrDefault("WS_ENABLED", "true") == "true",
			MaxConnections:  envOrDefaultInt("WS_MAX_CONNECTIONS", 1000),
			PingIntervalSec: envOrDefaultInt("WS_PING_INTERVAL_SECONDS", 30),
			ReadBufferSize:  envOrDefaultInt("WS_READ_BUFFER_SIZE", 1024),
			WriteBufferSize: envOrDefaultInt("WS_WRITE_BUFFER_SIZE", 1024),
		},
		Log: LogConfig{
			Level:  envOrDefault("LOG_LEVEL", "info"),
			Format: envOrDefault("LOG_FORMAT", "json"),
		},
		Classifier: ClassifierConfig{
			ReviewThreshold: envOrDefaultFloat("CLASSIFIER_REVIEW_THRESHOLD", 0.60),
			URL:             envOrDefault("CLASSIFIER_URL", ""),
			Timeout:         time.Duration(envOrDefaultInt("CLASSIFIER_TIMEOUT_MS", 5000)) * time.Millisecond,
			MaxRetries:      envOrDefaultInt("CLASSIFIER_MAX_RETRIES", 2),
			FallbackToStub:  envOrDefault("CLASSIFIER_FALLBACK_TO_STUB", "true") == "true",
		},
	}
}

func envOrDefaultFloat(key string, defaultVal float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func envOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func envOrDefaultInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func envOrDefaultInt32(key string, defaultVal int32) int32 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil {
			return int32(n)
		}
	}
	return defaultVal
}

func envOrDefaultInt64(key string, defaultVal int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return defaultVal
}

func envOrDefaultSlice(key string, defaultVal []string) []string {
	if val := os.Getenv(key); val != "" {
		parts := strings.Split(val, ",")
		var result []string
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultVal
}
