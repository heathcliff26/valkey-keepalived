package failoverclient

import (
	"context"
	"syscall"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	testutils "github.com/heathcliff26/valkey-keepalived/tests/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"
)

const (
	waitTimeout    = 30 * time.Second
	checkIntervall = time.Second
)

func TestClientBasicFailover(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	ctx := t.Context()

	setup, c := newSetupAndClient(t, "basic-failover", 3)
	go c.Run()
	t.Cleanup(func() {
		c.quit <- syscall.SIGTERM
	})

	require.Eventually(func() bool {
		return c.masterAddr != ""
	}, waitTimeout, checkIntervall, "Client should start and find current master")

	for i, n := range c.nodes {
		if i == 0 {
			assertNodeRoleEventually(t, ctx, n, master, "", i)
		} else {
			assertNodeRoleEventually(t, ctx, n, slave, c.masterAddr, i)
		}
	}

	oldMaster := c.masterAddr

	err := setup.StopNode(0)
	require.NoError(err, "Should stop the first node")

	assert.Eventually(func() bool {
		return c.masterAddr != oldMaster
	}, waitTimeout, checkIntervall, "Should fail over to new master")

	assertNodeDown(t, c.nodes[0], 0)

	assertNodeRoleEventually(t, ctx, c.nodes[1], master, "", 1)
	assertNodeRoleEventually(t, ctx, c.nodes[2], slave, c.masterAddr, 2)
}

func TestNodeRecoveryScenario(t *testing.T) {
	require := require.New(t)
	ctx := t.Context()

	setup, c := newSetupAndClient(t, "node-recovery", 3)

	err := setup.StopNode(1)
	require.NoError(err, "should stop node 1")

	go c.Run()
	t.Cleanup(func() {
		c.quit <- syscall.SIGTERM
	})

	require.Eventually(func() bool {
		return c.masterAddr != ""
	}, waitTimeout, checkIntervall, "Client should start and find current master")

	assertNodeRoleEventually(t, ctx, c.nodes[0], master, "", 0)

	assertNodeRoleEventually(t, ctx, c.nodes[2], slave, c.masterAddr, 2)

	err = setup.StartNode(1)
	require.NoError(err, "should start node 1")
	c.nodes[1].address, err = testutils.GetContainerIP(setup.Nodes[1])
	require.NoError(err, "Should get ip of node 1")

	require.Eventually(func() bool {
		return c.nodes[1].up
	}, waitTimeout, checkIntervall, "Node 1 should come back up")

	assertNodeRoleEventually(t, ctx, c.nodes[1], slave, c.masterAddr, 1)
}

func TestReplication(t *testing.T) {
	require := require.New(t)
	ctx := t.Context()

	_, c := newSetupAndClient(t, "replication", 3)
	go c.Run()
	t.Cleanup(func() {
		c.quit <- syscall.SIGTERM
	})

	require.Eventually(func() bool {
		return c.masterAddr != ""
	}, waitTimeout, checkIntervall, "Client should start and find current master")

	k, v := "testreplicationkey", "testreplicationvalue"

	err := c.nodes[0].client.Do(ctx, c.nodes[0].client.B().Set().Key(k).Value(v).Build()).Error()
	require.NoError(err, "Should write value")

	for i, n := range c.nodes {
		require.Eventuallyf(func() bool {
			res, err := n.client.Do(ctx, n.client.B().Get().Key(k).Build()).ToString()
			if err != nil {
				t.Logf("Failed to get key from node %d: %v", i, err)
			}
			return res == v
		}, waitTimeout, checkIntervall, "Node %d should have the key value pair", i)
	}
}

func TestClientClose(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	nodes := make([]*node, 3)
	for i := range nodes {
		mr := miniredis.RunT(t)
		opt := valkey.ClientOption{
			InitAddress:  []string{mr.Addr()},
			DisableCache: true,
			DisableRetry: true,
		}
		client, err := valkey.NewClient(opt)
		require.NoError(err, "Should create valkey client")
		nodes[i] = &node{
			client: client,
		}
	}

	c := &FailoverClient{
		nodes: nodes,
	}

	c.Close()

	for i, n := range c.nodes {
		assert.Nilf(n.client, "Node %d should have closed the client", i)
	}
}

// Create a new test setup and failoverclient.
// Skip test if no container runtime is found.
// Ensure cleanup is called for the setup.
func newSetupAndClient(t *testing.T, prefix string, nodeCount int) (*testutils.FailoverSetup, *FailoverClient) {
	if !testutils.HasContainerRuntimer() {
		t.Skip("No container runtime found")
	}

	setup, virtualIP, nodes, err := testutils.NewFailoverSetup(prefix, nodeCount)
	require.NoError(t, err, "Should create setup")
	t.Cleanup(setup.Cleanup)

	cfg := ValkeyConfig{
		VirtualAddress: virtualIP,
		Port:           6379,
		Nodes:          nodes,
	}
	return setup, NewFailoverClient(cfg)
}

func getRoleOfNode(ctx context.Context, n *node) (string, string, error) {
	res, err := n.getReplicationInfo(ctx)
	if err != nil {
		return "", "", err
	}
	return ParseValueFromInfo(res, role), res, nil
}

func assertNodeDown(t *testing.T, n *node, id int) {
	assert.Falsef(t, n.up, "Node %d should be down", id)
	assert.Nil(t, n.client, "Node %d should not have a client", id)
}

func assertNodeRoleEventually(t *testing.T, ctx context.Context, n *node, expectedRole, masterAddr string, id int) {
	assert.Eventuallyf(t, func() bool {
		role, info, err := getRoleOfNode(ctx, n)
		if err != nil {
			t.Logf("Failed to get role of node %d: %v", id, err)
			return false
		}

		if role != expectedRole {
			return false
		}
		if masterAddr != "" && masterAddr != ParseValueFromInfo(info, masterHost) {
			t.Logf("Node %d has the wrong master, expected \"%s\" but has \"%s\"", id, masterAddr, ParseValueFromInfo(info, masterHost))
			return false
		}
		return true
	}, waitTimeout, checkIntervall, "Node %d should have the expected role %s", id, expectedRole)
}
