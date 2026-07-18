package conf

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
	// Schema isolates this service to a specific database schema (Phase 2.4).
	// Empty (the default) keeps the legacy behaviour: the connection uses
	// whatever database the DSN points at, so all services share one DB.
	// When set, xdb.Open rewrites the MySQL DBName or applies a Postgres
	// search_path so every statement resolves tables in this schema.
	Schema string `json:"schema,omitempty"`
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
