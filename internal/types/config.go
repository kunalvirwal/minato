package types

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

// config.yaml file is parsed to Config struct
type Config struct {
	Services []Service `yaml:"services"`
}
