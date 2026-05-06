package config

import appregistry "micro-one-api/internal/pkg/registry"

// Config holds the log-service configuration.
type Config struct {
	Server   ServerConfig       `json:"server"`
	Data     DataConfig         `json:"data"`
	Log      LogSVCConfig       `json:"log_svc"`
	Registry appregistry.Config `json:"registry"`
}

type ServerConfig struct {
	HTTP HTTPConfig `json:"http"`
	GRPC GRPCConfig `json:"grpc"`
}

type HTTPConfig struct {
	Addr string `json:"addr"`
}

type GRPCConfig struct {
	Addr string `json:"addr"`
}

type DataConfig struct {
	Database DatabaseConfig `json:"database"`
	Redis    RedisConfig    `json:"redis"`
}

type DatabaseConfig struct {
	Driver string `json:"driver"`
	Source string `json:"source"`
}

type RedisConfig struct {
	Addr string `json:"addr"`
}

// LogSVCConfig holds log-service-specific settings.
type LogSVCConfig struct {
	// RetentionDays is how many days to retain logs.
	RetentionDays int `json:"retention_days"`
	// BatchSize is the batch insert size for log ingestion.
	BatchSize int `json:"batch_size"`
}
