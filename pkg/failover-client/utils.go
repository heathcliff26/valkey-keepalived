package failoverclient

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/valkey-io/valkey-go"
)

func newValkeyClient(addr string, port int64, option valkey.ClientOption) (valkey.Client, error) {
	option.InitAddress = []string{fmt.Sprintf("%s:%d", addr, port)}
	return valkey.NewClient(option)
}

// Takes a given info result from valkey and extracts the wanted value
func ParseValueFromInfo(info string, key string) string {
	fields := strings.Split(info, "\r\n")

	for _, field := range fields {
		keyval := strings.SplitN(field, ":", 2)
		if len(keyval) != 2 {
			continue
		}
		if keyval[0] == key {
			return keyval[1]
		}
	}

	slog.Error("Could not find the requested key in info", "info", info, "key", key)
	return ""
}
