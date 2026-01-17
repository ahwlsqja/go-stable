package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Chain    ChainConfig
	Worker   WorkerConfig
}

type ServerConfig struct {
	Host         string        `envconfig:"SERVER_HOST" default:"0.0.0.0"`
	Port         int           `envconfig:"SERVER_PORT" default:"8080"`
	ReadTimeout  time.Duration `envconfig:"SERVER_READ_TIMEOUT" default:"10s"`
	WriteTimeout time.Duration `envconfig:"SERVER_WRITE_TIMEOUT" default:"30s"`
	Environment  string        `envconfig:"ENVIRONMENT" default:"development"`
}

func (s ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

type DatabaseConfig struct {
	Host            string        `envconfig:"DB_HOST" default:"localhost"`
	Port            int           `envconfig:"DB_PORT" default:"3306"`
	User            string        `envconfig:"DB_USER" default:"app"`
	Password        string        `envconfig:"DB_PASSWORD" default:"apppassword"`
	Name            string        `envconfig:"DB_NAME" default:"go_stable"`
	MaxOpenConns    int           `envconfig:"DB_MAX_OPEN_CONNS" default:"25"`
	MaxIdleConns    int           `envconfig:"DB_MAX_IDLE_CONNS" default:"5"`
	ConnMaxLifetime time.Duration `envconfig:"DB_CONN_MAX_LIFETIME" default:"5m"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
		d.User, d.Password, d.Host, d.Port, d.Name)
}

type RedisConfig struct {
	Host     string `envconfig:"REDIS_HOST" default:"localhost"`
	Port     int    `envconfig:"REDIS_PORT" default:"6379"`
	Password string `envconfig:"REDIS_PASSWORD" default:""`
	DB       int    `envconfig:"REDIS_DB" default:"0"`
}

func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type ChainConfig struct {
	RPCURL              string        `envconfig:"CHAIN_RPC_URL" default:"http://localhost:8545"`
	TokenAddress        string        `envconfig:"TOKEN_ADDRESS" default:""`
	MinterPrivateKey    string        `envconfig:"MINTER_PRIVATE_KEY" default:""`
	RequiredConfirms    int           `envconfig:"REQUIRED_CONFIRMS" default:"3"`
	TxTimeout           time.Duration `envconfig:"CHAIN_TX_TIMEOUT" default:"2m"`
	PollingInterval     time.Duration `envconfig:"CHAIN_POLLING_INTERVAL" default:"1s"`
}

type WorkerConfig struct {
	PollInterval    time.Duration `envconfig:"WORKER_POLL_INTERVAL" default:"5s"`
	BatchSize       int           `envconfig:"WORKER_BATCH_SIZE" default:"10"`
	MaxRetries      int           `envconfig:"WORKER_MAX_RETRIES" default:"5"`
	RetryBaseDelay  time.Duration `envconfig:"WORKER_RETRY_BASE_DELAY" default:"1s"`
	LockTTL         time.Duration `envconfig:"WORKER_LOCK_TTL" default:"30s"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &cfg, nil
}
