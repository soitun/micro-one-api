package registry

import (
	"fmt"
	"net"
	"time"

	"github.com/go-kratos/kratos/v2/registry"
	consulregistry "github.com/go-kratos/kratos/contrib/registry/consul/v2"
	consulapi "github.com/hashicorp/consul/api"
)

// Config holds registry configuration.
type Config struct {
	// Type of registry: "consul" or "" (empty = no registry, use static endpoints).
	Type string `json:"type"`
	// Consul configuration.
	Consul ConsulConfig `json:"consul"`
}

type ConsulConfig struct {
	Address             string `json:"address"`
	HealthCheckInterval int    `json:"health_check_interval"` // seconds
}

// NewRegistrar creates a Registrar based on config. Returns nil if no registry type is set.
func NewRegistrar(cfg Config) (registry.Registrar, error) {
	switch cfg.Type {
	case "consul":
		return newConsulRegistry(cfg.Consul)
	case "":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported registry type: %s", cfg.Type)
	}
}

// NewDiscovery creates a Discovery based on config. Returns nil if no registry type is set.
func NewDiscovery(cfg Config) (registry.Discovery, error) {
	switch cfg.Type {
	case "consul":
		return newConsulRegistry(cfg.Consul)
	case "":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported registry type: %s", cfg.Type)
	}
}

func newConsulRegistry(cfg ConsulConfig) (*consulregistry.Registry, error) {
	consulCfg := consulapi.DefaultConfig()
	if cfg.Address != "" {
		consulCfg.Address = cfg.Address
	}
	client, err := consulapi.NewClient(consulCfg)
	if err != nil {
		return nil, fmt.Errorf("consul client: %w", err)
	}

	opts := []consulregistry.Option{
		consulregistry.WithHealthCheck(true),
		consulregistry.WithHeartbeat(true),
	}
	if cfg.HealthCheckInterval > 0 {
		opts = append(opts, consulregistry.WithHealthCheckInterval(cfg.HealthCheckInterval))
	}

	return consulregistry.New(client, opts...), nil
}

// ResolveHost returns the preferred outbound IP for service registration.
func ResolveHost() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

// ParseDuration parses a duration string with a default fallback.
func ParseDuration(s string, defaultVal time.Duration) time.Duration {
	if s == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultVal
	}
	return d
}
