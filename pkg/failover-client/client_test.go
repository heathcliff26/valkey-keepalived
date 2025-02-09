package failoverclient

import (
	"syscall"
	"testing"
	"time"

	testutils "github.com/heathcliff26/valkey-keepalived/tests/utils"
	"github.com/stretchr/testify/assert"
)

func TestClientBasic(t *testing.T) {
	if !testutils.HasContainerRuntimer() {
		t.Skip("No container runtime found")
	}
	assert := assert.New(t)

	setup, virtualIP, nodes, err := testutils.NewFailoverSetup("basic", 3)
	if !assert.NoError(err, "Should create setup") {
		t.FailNow()
	}
	t.Cleanup(setup.Cleanup)

	cfg := ValkeyConfig{
		VirtualAddress: virtualIP,
		Port:           6379,
		Nodes:          nodes,
	}
	c := NewFailoverClient(cfg)
	go c.Run()
	t.Cleanup(func() {
		c.quit <- syscall.SIGTERM
	})

	assert.Eventually(func() bool {
		return c.currentMaster != ""
	}, 10*time.Second, time.Second, "Client should start and find current master")

	ctx := t.Context()

	for i, n := range c.nodes {
		if !assert.NotNilf(n.client, "Node %d should have a client", i) {
			continue
		}
		res, err := n.client.Do(ctx, n.client.B().Info().Section("replication").Build()).ToString()
		assert.NoErrorf(err, "Should receive info from node number %d", i)
		expectedRole := "slave"
		if i == 0 {
			expectedRole = "master"
		}
		assert.Equalf(expectedRole, parseValueFromInfo(res, "role"), "Node %d should have the expected role", i)
	}
}
