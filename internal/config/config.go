package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Deployment DeploymentConfig `mapstructure:"deployment"`
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Logging    LoggingConfig    `mapstructure:"logging"`
	CORS       CORSConfig       `mapstructure:"cors"`
	External   ExternalConfig   `mapstructure:"external"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
}

type DeploymentConfig struct {
	Mode string `mapstructure:"mode"` // single or separate
}

type ServerConfig struct {
	HTTPPort int `mapstructure:"http_port"`
	GRPCPort int `mapstructure:"grpc_port"`
}

type DatabaseConfig struct {
	URL            string `mapstructure:"url"`
	MaxConnections int    `mapstructure:"max_connections"`
	IdleConnections int   `mapstructure:"idle_connections"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type CORSConfig struct {
	Origins []string `mapstructure:"origins"`
	Methods []string `mapstructure:"methods"`
	Headers []string `mapstructure:"headers"`
}

type ExternalConfig struct {
	DatabaseGRPCURL string `mapstructure:"database_grpc_url"`
	HandlersGRPCURL string `mapstructure:"handlers_grpc_url"`
	DBServiceURL    string `mapstructure:"db_service_url"`
}

type RedisConfig struct {
	URL      string `mapstructure:"url"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type MonitoringConfig struct {
	MetricsEnabled bool `mapstructure:"metrics_enabled"`
	MetricsPort    int  `mapstructure:"metrics_port"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set defaults
	setDefaults()

	// Bind environment variables
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read config file if exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

func setDefaults() {
	// Deployment defaults
	viper.SetDefault("deployment.mode", "single")

	// Server defaults
	viper.SetDefault("server.http_port", 8080)
	viper.SetDefault("server.grpc_port", 9090)

	// Database defaults
	viper.SetDefault("database.url", "postgres://user:pass@localhost:5432/etcmeisai")
	viper.SetDefault("database.max_connections", 25)
	viper.SetDefault("database.idle_connections", 5)

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")

	// CORS defaults
	viper.SetDefault("cors.origins", []string{"*"})
	viper.SetDefault("cors.methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	viper.SetDefault("cors.headers", []string{"Content-Type", "Authorization"})

	// External service defaults
	viper.SetDefault("external.database_grpc_url", "localhost:50051")
	viper.SetDefault("external.handlers_grpc_url", "localhost:50052")

	// Redis defaults
	viper.SetDefault("redis.url", "redis://localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// Monitoring defaults
	viper.SetDefault("monitoring.metrics_enabled", true)
	viper.SetDefault("monitoring.metrics_port", 9091)
}

func validate(cfg *Config) error {
	if cfg.Deployment.Mode != "single" && cfg.Deployment.Mode != "separate" {
		return fmt.Errorf("invalid deployment mode: %s", cfg.Deployment.Mode)
	}

	if cfg.Server.HTTPPort <= 0 || cfg.Server.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", cfg.Server.HTTPPort)
	}

	if cfg.Server.GRPCPort <= 0 || cfg.Server.GRPCPort > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", cfg.Server.GRPCPort)
	}

	return nil
}

func (c *Config) IsSingleMode() bool {
	return c.Deployment.Mode == "single"
}

func (c *Config) IsSeparateMode() bool {
	return c.Deployment.Mode == "separate"
}