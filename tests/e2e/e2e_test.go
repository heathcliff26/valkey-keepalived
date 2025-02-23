package e2e

import (
	"fmt"
	"os"
	"testing"
	"time"

	failoverclient "github.com/heathcliff26/valkey-keepalived/pkg/failover-client"
	"github.com/heathcliff26/valkey-keepalived/tests/utils"
	"github.com/stretchr/testify/assert"
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
  virtualAddress: "%s"
  nodes:
    - "%s"
    - "%s"
`

func TestE2E(t *testing.T) {
	assert := assert.New(t)
	ctx := t.Context()

	err := utils.ExecCRI("build", "-t", containerImage, "../..")
	if !assert.NoError(err, "Should build container image") {
		t.FailNow()
	}

	setup, virtualAddress, nodes, err := utils.NewFailoverSetup("e2e", 2)
	if !assert.NoError(err, "Should create test setup") {
		t.FailNow()
	}
	t.Cleanup(setup.Cleanup)

	cfg := fmt.Sprintf(valkeyKeepalivedConfigTemplate, virtualAddress, nodes[0], nodes[1])
	cfgFile, err := os.CreateTemp("", "test-e2e-*.yaml")
	if !assert.NoError(err, "Should create config file") {
		t.FailNow()
	}
	t.Cleanup(func() {
		cfgFile.Close()
		os.Remove(cfgFile.Name())
	})

	_, err = cfgFile.WriteString(cfg)
	if !assert.NoError(err, "Should write config to file") {
		t.FailNow()
	}
	err = cfgFile.Chmod(0644)
	if !assert.NoError(err, "Should add read permissions to config file") {
		t.FailNow()
	}

	err = utils.ExecCRI("run", "-d", "--rm", "--name", containerName, "-v", cfgFile.Name()+":/config/config.yaml:z", containerImage)
	if !assert.NoError(err, "Should start valkey-keepalived container") {
		t.FailNow()
	}
	t.Cleanup(func() {
		_ = utils.ExecCRI("stop", containerName)
	})

	for i, node := range nodes {
		opt := valkey.ClientOption{
			InitAddress:  []string{node + ":6379"},
			DisableCache: true,
			DisableRetry: true,
		}
		c, err := valkey.NewClient(opt)
		if !assert.NoErrorf(err, "Should create client for node %d", i) {
			t.FailNow()
		}
		ok := assert.Eventually(func() bool {
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
		if !ok {
			t.FailNow()
		}
	}
}
