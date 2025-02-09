package failoverclient

import "fmt"

type ValkeyConfig struct {
	VirtualAddress string   `json:"virtualAddress"`
	Port           int64    `json:"port,omitempty"`
	Nodes          []string `json:"nodes"`
	Username       string   `json:"username,omitempty"`
	Password       string   `json:"password,omitempty"`
	TLS            bool     `json:"tls,omitempty"`
}

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
