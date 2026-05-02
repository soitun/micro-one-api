package config

// Config holds the admin-api configuration.
type Config struct {
	Server  ServerConfig  `json:"server"`
	Clients ClientsConfig `json:"clients"`
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

type ClientsConfig struct {
	Identity identityClientConfig `json:"identity"`
	Channel  channelClientConfig `json:"channel"`
	Billing  billingClientConfig  `json:"billing"`
}

type identityClientConfig struct {
	Endpoint string `json:"endpoint"`
}

type channelClientConfig struct {
	Endpoint string `json:"endpoint"`
}

type billingClientConfig struct {
	Endpoint string `json:"endpoint"`
}
