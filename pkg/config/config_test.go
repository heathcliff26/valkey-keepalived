package config

import (
	"fmt"
	"log/slog"
	"reflect"
	"testing"

	failoverclient "github.com/heathcliff26/valkey-keepalived/pkg/failover-client"
	"github.com/stretchr/testify/assert"
)

func TestValidConfigs(t *testing.T) {
	c1 := Config{
		LogLevel: "debug",
		Valkey: failoverclient.ValkeyConfig{
			VirtualAddress: "10.8.0.10",
			Port:           6380,
			Nodes:          []string{"10.8.0.11", "10.8.0.12"},
			Username:       "testuser",
			Password:       "testpassword",
			TLS:            true,
		},
	}
	c2 := Config{
		LogLevel: DEFAULT_LOG_LEVEL,
		Valkey: failoverclient.ValkeyConfig{
			VirtualAddress: "10.8.0.10",
			Port:           DEFAULT_PORT,
			Nodes:          []string{"10.8.0.11", "10.8.0.12"},
		},
	}
	tMatrix := []struct {
		Name, Path string
		Result     Config
	}{
		{
			Name:   "ValidConfig",
			Path:   "testdata/valid-config.yaml",
			Result: c1,
		},
		{
			Name:   "ValidConfigWithDefaults",
			Path:   "testdata/valid-config-defaults.yaml",
			Result: c2,
		},
	}

	for _, tCase := range tMatrix {
		t.Run(tCase.Name, func(t *testing.T) {
			c, err := LoadConfig(tCase.Path, false)

			assert := assert.New(t)

			if !assert.Nil(err) {
				t.Fatalf("Failed to load config: %v", err)
			}
			assert.Equal(tCase.Result, c)
		})
	}
}

func TestInvalidConfig(t *testing.T) {
	tMatrix := []struct {
		Name, Path, Mode, Error string
	}{
		{
			Name:  "EmptyPath",
			Path:  "",
			Error: "*fs.PathError",
		},
		{
			Name:  "InvalidPath",
			Path:  "file-does-not-exist.yaml",
			Error: "*fs.PathError",
		},
		{
			Name:  "NotYaml",
			Path:  "testdata/not-a-config.txt",
			Error: "*yaml.TypeError",
		},
		{
			Name:  "InvalidLogLevel",
			Path:  "testdata/invalid-config-loglevel.yaml",
			Error: "*errors.errorString",
		},
		{
			Name:  "InvalidValkeyConfig",
			Path:  "testdata/invalid-config-valkey.yaml",
			Error: "*errors.errorString",
		},
	}

	for _, tCase := range tMatrix {
		t.Run(tCase.Name, func(t *testing.T) {
			_, err := LoadConfig(tCase.Path, false)

			if !assert.Error(t, err) {
				t.Fatal("Did not receive an error")
			}
			if !assert.Equal(t, tCase.Error, reflect.TypeOf(err).String()) {
				t.Fatalf("Received invalid error: %v", err)
			}
		})
	}
}

func TestEnvSubstitution(t *testing.T) {
	assert := assert.New(t)

	c := Config{
		LogLevel: "debug",
		Valkey: failoverclient.ValkeyConfig{
			VirtualAddress: "10.8.0.10",
			Port:           6380,
			Nodes:          []string{"10.8.0.11", "10.8.0.12"},
			Username:       "testuser",
			Password:       "testpassword",
			TLS:            true,
		},
	}
	t.Setenv("TESTUSERNAME", c.Valkey.Username)
	t.Setenv("TESTPASSWORD", c.Valkey.Password)

	res, err := LoadConfig("testdata/env-config.yaml", true)

	if !assert.Nil(err) {
		t.Fatalf("Could not load config: %v", err)
	}
	assert.Equal(c, res)
}

func TestSetLogLevel(t *testing.T) {
	tMatrix := []struct {
		Name  string
		Level slog.Level
		Error error
	}{
		{"debug", slog.LevelDebug, nil},
		{"info", slog.LevelInfo, nil},
		{"warn", slog.LevelWarn, nil},
		{"error", slog.LevelError, nil},
		{"DEBUG", slog.LevelDebug, nil},
		{"INFO", slog.LevelInfo, nil},
		{"WARN", slog.LevelWarn, nil},
		{"ERROR", slog.LevelError, nil},
		{"Unknown", 0, fmt.Errorf("unkown log level \"unknown\"")},
	}
	t.Cleanup(func() {
		err := setLogLevel(DEFAULT_LOG_LEVEL)
		if err != nil {
			t.Fatalf("Failed to cleanup after test: %v", err)
		}
	})

	for _, tCase := range tMatrix {
		t.Run(tCase.Name, func(t *testing.T) {
			err := setLogLevel(tCase.Name)

			assert := assert.New(t)

			if !assert.Equal(tCase.Error, err) {
				t.Fatalf("Received invalid error: %v", err)
			}
			if err == nil {
				assert.Equal(tCase.Level, logLevel.Level())
			}
		})
	}
}
