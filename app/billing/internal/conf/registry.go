package conf

import appregistry "micro-one-api/platform/registry"

// ToRegistryConfig converts the proto Registry to the platform registry Config.
func (r *Registry) ToRegistryConfig() appregistry.Config {
	cfg := appregistry.Config{
		Type: r.Type,
	}

	if r.Consul != nil {
		metadata := make(map[string]string)
		for k, v := range r.Metadata {
			metadata[k] = v
		}

		cfg.Consul = appregistry.ConsulConfig{
			Address:             r.Consul.Address,
			HealthCheckInterval: int(r.Consul.HealthCheckInterval),
			Metadata:            metadata,
		}
	}

	return cfg
}
