package config

import appregistry "micro-one-api/platform/registry"

// Config holds the config-service configuration.
type Config struct {
	Server   ServerConfig       `json:"server"`
	Data     DataConfig         `json:"data"`
	Config   ConfigSVCConfig    `json:"config_svc"`
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

// ConfigSVCConfig holds config-service-specific settings.
type ConfigSVCConfig struct {
	// CacheTTL is the TTL in seconds for cached config entries.
	CacheTTL int `json:"cache_ttl"`
	// Namespace is the default config namespace.
	Namespace string `json:"namespace"`
}
