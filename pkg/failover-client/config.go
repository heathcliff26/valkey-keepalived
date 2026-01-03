package failoverclient

import "fmt"

type ValkeyConfig struct {
	VirtualAddress string   `yaml:"virtualAddress"`
	Port           int64    `yaml:"port,omitempty"`
	Nodes          []string `yaml:"nodes"`
	Username       string   `yaml:"username,omitempty"`
	Password       string   `yaml:"password,omitempty"`
	TLS            bool     `yaml:"tls,omitempty"`
}

// Ensure that the given config is valid
func (c ValkeyConfig) Validate() error {
	if c.VirtualAddress == "" {
		return fmt.Errorf("missing virtual address")
	}
	if c.Port < 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port, needs to be between 0-65535")
	}
	if len(c.Nodes) < 1 {
		return fmt.Errorf("need to have at least 1 node listed")
	}

	return nil
}
