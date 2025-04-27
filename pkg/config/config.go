package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	failoverclient "github.com/heathcliff26/valkey-keepalived/pkg/failover-client"
	"sigs.k8s.io/yaml"
)

const (
	DEFAULT_CONFIG_PATH = "/config/config.yaml"

	DEFAULT_LOG_LEVEL = "info"
	DEFAULT_PORT      = 6379
)

var logLevel *slog.LevelVar

// Initialize the logger
func init() {
	logLevel = &slog.LevelVar{}
	opts := slog.HandlerOptions{
		Level: logLevel,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &opts))
	slog.SetDefault(logger)
}

type Config struct {
	LogLevel string                      `json:"logLevel,omitempty"`
	Valkey   failoverclient.ValkeyConfig `json:"valkey"`
}

// Returns a Config with default values set
func DefaultConfig() Config {
	return Config{
		LogLevel: DEFAULT_LOG_LEVEL,
		Valkey: failoverclient.ValkeyConfig{
			Port: DEFAULT_PORT,
		},
	}
}

// Loads config from file, returns error if config is invalid
// Arguments:
//
//	path: Path to config file
//	env: Determines if enviroment variables in the file will be expanded before decoding
func LoadConfig(path string, env bool) (Config, error) {
	c := DefaultConfig()

	if path == "" {
		path = DEFAULT_CONFIG_PATH
	}

	// #nosec G304: Local users can decide on the config file path freely.
	f, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	if env {
		f = []byte(os.ExpandEnv(string(f)))
	}

	err = yaml.Unmarshal(f, &c)
	if err != nil {
		return Config{}, err
	}

	err = setLogLevel(c.LogLevel)
	if err != nil {
		return Config{}, err
	}

	err = c.Valkey.Validate()
	if err != nil {
		return Config{}, err
	}

	return c, nil
}

// Parse a given string and set the resulting log level
func setLogLevel(level string) error {
	switch strings.ToLower(level) {
	case "debug":
		logLevel.Set(slog.LevelDebug)
	case "info":
		logLevel.Set(slog.LevelInfo)
	case "warn":
		logLevel.Set(slog.LevelWarn)
	case "error":
		logLevel.Set(slog.LevelError)
	default:
		return fmt.Errorf("unkown log level \"%s\"", strings.ToLower(level))
	}
	return nil
}
