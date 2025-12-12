package e2e

import (
	"fmt"
	"os"
	"testing"
	"time"

	failoverclient "github.com/heathcliff26/valkey-keepalived/pkg/failover-client"
	"github.com/heathcliff26/valkey-keepalived/tests/utils"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"
)

const (
	containerImage = "localhost/valkey-keepalived:e2e"
	containerName  = "test-e2e-valkey-keepalived"
)

const (
	waitTimeout    = 30 * time.Second
	checkIntervall = time.Second
)

const valkeyKeepalivedConfigTemplate = `logLevel: debug
valkey:
  virtualAddress: "localhost"
  port: %d
  nodes:
    - "localhost:%d"
    - "localhost:%d"
`

func TestE2E(t *testing.T) {
	require := require.New(t)
	ctx := t.Context()

	err := utils.ExecCRI("build", "-t", containerImage, "../..")
	require.NoError(err, "Should build container image")

	setup, err := utils.NewFailoverSetup("e2e", 2)
	require.NoError(err, "Should create test setup")
	t.Cleanup(setup.Cleanup)

	cfg := fmt.Sprintf(valkeyKeepalivedConfigTemplate, setup.Port, setup.Nodes[0].Port, setup.Nodes[1].Port)
	cfgFile, err := os.CreateTemp("", "test-e2e-*.yaml")
	require.NoError(err, "Should create config file")
	t.Cleanup(func() {
		cfgFile.Close()
		os.Remove(cfgFile.Name())
	})

	_, err = cfgFile.WriteString(cfg)
	require.NoError(err, "Should write config to file")
	err = cfgFile.Chmod(0644)
	require.NoError(err, "Should add read permissions to config file")

	err = utils.ExecCRI("run", "-d", "--rm", "--name", containerName, "-v", cfgFile.Name()+":/config/config.yaml:z", "--net", "host", containerImage)
	require.NoError(err, "Should start valkey-keepalived container")
	t.Cleanup(func() {
		_ = utils.ExecCRI("stop", containerName)
	})

	for i, node := range setup.Nodes {
		opt := valkey.ClientOption{
			InitAddress:  []string{fmt.Sprintf("127.0.0.1:%d", node.Port)},
			DisableCache: true,
			DisableRetry: true,
		}
		c, err := valkey.NewClient(opt)
		require.NoErrorf(err, "Should create client for node %d", i)

		require.Eventually(func() bool {
			res, err := c.Do(ctx, c.B().Info().Section("replication").Build()).ToString()
			if err != nil {
				t.Logf("Failed to connect to node %d: %v", i, err)
				return false
			}
			expectedRole := "slave"
			if i == 0 {
				expectedRole = "master"
			}

			role := failoverclient.ParseValueFromInfo(res, "role")
			if expectedRole != role {
				t.Logf("Node %d has role \"%s\" but should have \"%s\"", i, role, expectedRole)
				return false
			}
			return true
		}, waitTimeout, checkIntervall, "Should connect to node %d and verify that it has the expected role")
	}
}
