package failoverclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigValidate(t *testing.T) {
	tMatrix := []struct {
		Name   string
		Config ValkeyConfig
		Valid  bool
	}{
		{
			Name: "ValidConfig-1",
			Config: ValkeyConfig{
				VirtualAddress: "10.8.0.10",
				Port:           6379,
				Nodes:          []string{"10.8.0.11", "10.8.0.12"},
			},
			Valid: true,
		},
		{
			Name: "ValidConfig-2",
			Config: ValkeyConfig{
				VirtualAddress: "10.8.0.10",
				Port:           6379,
				Nodes:          []string{"10.8.0.11", "10.8.0.12"},
				Username:       "testuser",
				Password:       "testpassword",
				TLS:            true,
			},
			Valid: true,
		},
		{
			Name: "MissingVirtualAddress",
			Config: ValkeyConfig{
				VirtualAddress: "",
				Port:           6379,
				Nodes:          []string{"10.8.0.11", "10.8.0.12"},
			},
			Valid: false,
		},
		{
			Name: "MissingNodes",
			Config: ValkeyConfig{
				VirtualAddress: "10.8.0.10",
				Port:           6379,
			},
			Valid: false,
		},
		{
			Name: "NegativePortNumber",
			Config: ValkeyConfig{
				VirtualAddress: "10.8.0.10",
				Port:           -1,
				Nodes:          []string{"10.8.0.11", "10.8.0.12"},
			},
			Valid: false,
		},
		{
			Name: "PortToBig",
			Config: ValkeyConfig{
				VirtualAddress: "10.8.0.10",
				Port:           65536,
				Nodes:          []string{"10.8.0.11", "10.8.0.12"},
			},
			Valid: false,
		},
	}

	for _, tCase := range tMatrix {
		t.Run(tCase.Name, func(t *testing.T) {
			assert := assert.New(t)

			if tCase.Valid {
				assert.NoError(tCase.Config.Validate())
			} else {
				assert.Error(tCase.Config.Validate())
			}
		})
	}
}
