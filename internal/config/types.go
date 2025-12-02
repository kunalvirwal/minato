package config

type Upstream struct {
	Host       string `yaml:"host"`
	Health_uri string `yaml:"health_uri"`
}

// Services are the Load Balancers we have to create which are defined in Config.yaml
// Host here refers to complete inbound URL including path prefix and has been used for generalization
type Service struct {
	Name      string     `yaml:"name"`
	Port      int        `yaml:"listen_port"`
	Balancer  string     `yaml:"balancer"`
	Hosts     []string   `yaml:"hosts"`
	Upstreams []Upstream `yaml:"upstreams"`
}

type Cache struct {
	Enabled  bool   `yaml:"enabled"`
	MaxSize  uint64 `yaml:"max_size"`
	Capacity uint64 `yaml:"capacity"`
	Type     string `yaml:"type"`
	TTL      uint64 `yaml:"ttl"`
}

// config.yaml file is parsed to Config struct
type Config struct {
	Cache    Cache     `yaml:"cache"`
	Services []Service `yaml:"services"`
}
