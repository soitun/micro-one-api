package config

// Config holds the monitor-worker configuration.
type Config struct {
	Server  ServerConfig     `json:"server"`
	Data    DataConfig       `json:"data"`
	Monitor MonitorSVCConfig `json:"monitor_svc"`
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

// MonitorSVCConfig holds monitor-worker-specific settings.
type MonitorSVCConfig struct {
	// CollectIntervalSec is the metrics collection interval in seconds.
	CollectIntervalSec int `json:"collect_interval_sec"`
	// AlertRetentionDays is how many days to keep alert history.
	AlertRetentionDays int `json:"alert_retention_days"`
	// NotifyEmail is the recipient for alert notifications.
	NotifyEmail string `json:"notify_email"`
}
