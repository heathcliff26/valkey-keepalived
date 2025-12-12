package failoverclient

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"
)

func TestNodeConnect(t *testing.T) {
	assert := assert.New(t)
	mr := miniredis.RunT(t)

	n := &node{
		address: mr.Host(),
		port:    int64(mr.Server().Addr().Port),
	}
	err := n.connect(t.Context(), valkey.ClientOption{
		DisableCache: true,
		DisableRetry: true,
	})

	assert.Equal("section (server) is not supported", err.Error(), "Should return miniredis error")
	assert.Nil(n.client, "Should not set client")
	assert.False(n.up, "Should not mark node as up")
	assert.Empty(n.runID, "Should have no run_id")
}

func TestNodePing(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	ctx := t.Context()

	mr, n, err := newNodeWithMiniredis(t)
	require.NoError(err, "Should create node with client")

	n.ping(ctx)
	assert.NotNil(n.client, "Node should retain client")
	assert.True(n.up, "Node should be up")

	mr.Close()

	n.ping(ctx)
	assert.Nil(n.client, "Node should close client")
	assert.False(n.up, "Node should be down")
}

func TestNodeMaster(t *testing.T) {
	t.Run("NoClient", func(t *testing.T) {
		assert.NotPanics(t, func() {
			_ = (&node{}).master(t.Context())
		}, "Should not panic when client is nil")
	})
	t.Run("CacheHit", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		mr, n, err := newNodeWithMiniredis(t)
		require.NoError(err, "Should create node with client")

		mr.Close()
		n.roleCache.Save(master, nil)

		assert.Nil(n.master(t.Context()), "Should hit cache and not attempt to connect to server")
	})
	t.Run("CacheExpired", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		mr, n, err := newNodeWithMiniredis(t)
		require.NoError(err, "Should create node with client")

		mr.Close()
		n.roleCache.Save(master, nil)
		n.roleCache.expire = time.Now().Add(-time.Minute)

		assert.Error(n.master(t.Context()), "Should not hit the cache and error out instead")
	})
	t.Run("CacheEmpty", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		mr, n, err := newNodeWithMiniredis(t)
		require.NoError(err, "Should create node with client")

		mr.Close()

		assert.Error(n.master(t.Context()), "Should not hit the cache and error out instead")
		assert.Empty(n.roleCache, "Should not save to cache on error")
	})
}

func TestNodeSlave(t *testing.T) {
	t.Run("NoClient", func(t *testing.T) {
		assert.NotPanics(t, func() {
			_ = (&node{}).slave(t.Context(), nil)
		}, "Should not panic when client is nil")
	})
	t.Run("MasterNil", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		mr, n, err := newNodeWithMiniredis(t)
		require.NoError(err, "Should create node with client")
		mr.Close()

		n.roleCache.Save(slave, &node{address: "testmaster", port: 6379})
		assert.NotPanics(func() {
			_ = n.slave(t.Context(), nil)
		}, "Should not panic when new master is nil")
	})
	t.Run("CacheHit", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		mr, n, err := newNodeWithMiniredis(t)
		require.NoError(err, "Should create node with client")

		mr.Close()
		n.roleCache.Save(slave, &node{})

		assert.Nil(n.slave(t.Context(), &node{}), "Should hit cache and not attempt to connect to server")
	})
	t.Run("CacheExpired", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		mr, n, err := newNodeWithMiniredis(t)
		require.NoError(err, "Should create node with client")

		mr.Close()
		n.roleCache.Save(slave, &node{})
		n.roleCache.expire = time.Now().Add(-time.Minute)

		assert.Error(n.slave(t.Context(), &node{}), "Should not hit the cache and error out instead")
	})
	t.Run("CacheEmpty", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		mr, n, err := newNodeWithMiniredis(t)
		require.NoError(err, "Should create node with client")

		mr.Close()

		assert.Error(n.slave(t.Context(), &node{}), "Should not hit the cache and error out instead")
		assert.Empty(n.roleCache, "Should not save to cache on error")
	})
}

func TestNodeCacheSave(t *testing.T) {
	_, c := newSetupAndClient(t, "node-cache-save", 2)

	for i, n := range c.nodes {
		err := n.connect(t.Context(), valkey.ClientOption{})
		require.NoErrorf(t, err, "Should connect to node %d", i)
	}

	t.Run("Master", func(t *testing.T) {
		assert := assert.New(t)
		n := c.nodes[0]

		err := n.master(t.Context())

		assert.NoError(err, "Should set node to master")
		assert.Equal(master, n.roleCache.role, "Should save role in cache")
		assert.NotEmpty(n.roleCache.expire, "Should set expire time")
		assert.False(n.roleCache.isExpired(), "Cache should not be expired")
	})
	t.Run("Slave", func(t *testing.T) {
		assert := assert.New(t)
		n := c.nodes[1]

		err := n.slave(t.Context(), c.nodes[0])

		assert.NoError(err, "Should set node to slave")
		assert.Equal(slave, n.roleCache.role, "Should save role in cache")
		assert.Equal(c.nodes[0], n.roleCache.master, "Should save master_host in cache")
		assert.NotEmpty(n.roleCache.expire, "Should set expire time")
		assert.False(n.roleCache.isExpired(), "Cache should not be expired")
	})
}

func newNodeWithMiniredis(t *testing.T) (*miniredis.Miniredis, *node, error) {
	mr := miniredis.RunT(t)

	opt := valkey.ClientOption{
		InitAddress:  []string{mr.Addr()},
		DisableCache: true,
		DisableRetry: true,
	}
	client, err := valkey.NewClient(opt)
	if err != nil {
		return nil, nil, err
	}

	n := &node{
		address:   mr.Host(),
		port:      int64(mr.Server().Addr().Port),
		client:    client,
		roleCache: &roleCache{},
	}

	return mr, n, nil
}
