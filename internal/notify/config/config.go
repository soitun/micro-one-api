package config

// Config holds the notify-worker configuration.
type Config struct {
	Server ServerConfig    `json:"server"`
	Data   DataConfig      `json:"data"`
	Notify NotifySVCConfig `json:"notify_svc"`
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

// NotifySVCConfig holds notify-worker-specific settings.
type NotifySVCConfig struct {
	// WebhookURL is the default webhook endpoint for notifications.
	WebhookURL string `json:"webhook_url"`
	// SMTPHost is the SMTP server host for email notifications.
	SMTPHost string `json:"smtp_host"`
	// SMTPPort is the SMTP server port.
	SMTPPort int `json:"smtp_port"`
	// SMTPUser is the SMTP auth username.
	SMTPUser string `json:"smtp_user"`
	// SMTPFrom is the sender email address.
	SMTPFrom string `json:"smtp_from"`
}
