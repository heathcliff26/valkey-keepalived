package failoverclient

import (
	"context"
	"fmt"
	"syscall"
	"testing"
	"time"

	testutils "github.com/heathcliff26/valkey-keepalived/tests/utils"
	"github.com/stretchr/testify/assert"
)

const (
	master = "master"
	slave  = "slave"
)

const (
	waitTimeout    = 30 * time.Second
	checkIntervall = time.Second
)

func TestClientBasicFailover(t *testing.T) {
	assert := assert.New(t)
	ctx := t.Context()

	setup, c := newSetupAndClient(t, "basic-failover", 3)
	go c.Run()
	t.Cleanup(func() {
		c.quit <- syscall.SIGTERM
	})

	assert.Eventually(func() bool {
		return c.currentMaster != ""
	}, waitTimeout, checkIntervall, "Client should start and find current master")

	for i, n := range c.nodes {
		role, _, err := getRoleOfNode(ctx, n)
		assert.NoErrorf(err, "Should not fail to get role of node %d", i)
		expectedRole := slave
		if i == 0 {
			expectedRole = master
		}
		assert.Equalf(expectedRole, role, "Node %d should have the expected role", i)
	}

	oldMaster := c.currentMaster

	err := setup.StopNode(0)
	if !assert.NoError(err, "Should stop the first node") {
		t.FailNow()
	}

	assert.Eventually(func() bool {
		return c.currentMaster != oldMaster
	}, waitTimeout, checkIntervall, "Should fail over to new master")

	assertNodeDown(assert, c.nodes[0], 0)

	role, _, err := getRoleOfNode(ctx, c.nodes[1])
	assert.NoError(err, "Should not fail to get role of node 1")
	assert.Equal(master, role, "Node 1 should be the new master")

	role, info, err := getRoleOfNode(ctx, c.nodes[2])
	assert.NoError(err, "Should not fail to get role of node 2")
	assert.Equal(slave, role, "Node 2 should still be a slave")
	assert.Equal(c.currentMaster, parseValueFromInfo(info, "master_host"), "Node 2 should have the correct master")
}

func TestNodeRecoveryScenario(t *testing.T) {
	assert := assert.New(t)
	ctx := t.Context()

	setup, c := newSetupAndClient(t, "node-recovery", 3)

	err := setup.StopNode(1)
	if !assert.NoError(err, "should stop node 1") {
		t.FailNow()
	}

	go c.Run()
	t.Cleanup(func() {
		c.quit <- syscall.SIGTERM
	})

	assert.Eventually(func() bool {
		return c.currentMaster != ""
	}, waitTimeout, checkIntervall, "Client should start and find current master")

	role, _, err := getRoleOfNode(ctx, c.nodes[0])
	assert.NoError(err, "Should not fail to get role of node 0")
	assert.Equal(master, role, "Node 0 should be the master")

	role, _, err = getRoleOfNode(ctx, c.nodes[2])
	assert.NoError(err, "Should not fail to get role of node 2")
	assert.Equal(slave, role, "Node 2 should be a slave")

	err = setup.StartNode(1)
	if !assert.NoError(err, "should start node 1") {
		t.FailNow()
	}
	c.nodes[1].address, err = testutils.GetContainerIP(setup.Nodes[1])
	if !assert.NoError(err, "Should get ip of node 1") {
		t.FailNow()
	}

	if !assert.Eventually(func() bool {
		return c.nodes[1].up
	}, waitTimeout, checkIntervall, "Node 1 should come back up") {
		t.FailNow()
	}

	role, info, err := getRoleOfNode(ctx, c.nodes[1])
	assert.NoError(err, "Should not fail to get role of node 1")
	assert.Equal(slave, role, "Node 1 should be a slave")
	assert.Equal(c.currentMaster, parseValueFromInfo(info, "master_host"), "Node 1 should have the correct master")
}

// Create a new test setup and failoverclient.
// Run test in parallel and skip if no container runtime is found.
// Ensure cleanup is called for the setup.
func newSetupAndClient(t *testing.T, prefix string, nodeCount int) (*testutils.FailoverSetup, *FailoverClient) {
	if !testutils.HasContainerRuntimer() {
		t.Skip("No container runtime found")
	}
	t.Parallel()

	setup, virtualIP, nodes, err := testutils.NewFailoverSetup(prefix, nodeCount)
	if !assert.NoError(t, err, "Should create setup") {
		t.FailNow()
	}
	t.Cleanup(setup.Cleanup)

	cfg := ValkeyConfig{
		VirtualAddress: virtualIP,
		Port:           6379,
		Nodes:          nodes,
	}
	return setup, NewFailoverClient(cfg)
}

func getRoleOfNode(ctx context.Context, n *node) (string, string, error) {
	if n.client == nil {
		return "", "", fmt.Errorf("node has no client")
	}
	res, err := n.client.Do(ctx, n.client.B().Info().Section("replication").Build()).ToString()
	if err != nil {
		return "", "", err
	}
	return parseValueFromInfo(res, "role"), res, nil
}

func assertNodeDown(assert *assert.Assertions, n *node, id int) {
	assert.Falsef(n.up, "Node %d should be down", id)
	assert.Nil(n.client, "Node %d should not have a client", id)
}
