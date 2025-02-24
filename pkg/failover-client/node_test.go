package failoverclient

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
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
	ctx := t.Context()

	mr, n, err := newNodeWithMiniredis(t)
	if !assert.NoError(err, "Should create node with client") {
		t.FailNow()
	}

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
		assert.Error(t, (&node{}).master(t.Context()), "Should not panic when client is nil")
	})
	t.Run("CacheHit", func(t *testing.T) {
		assert := assert.New(t)

		mr, n, err := newNodeWithMiniredis(t)
		if !assert.NoError(err, "Should create node with client") {
			t.FailNow()
		}

		mr.Close()
		n.roleCache.Save(master, "")

		assert.Nil(n.master(t.Context()), "Should hit cache and not attempt to connect to server")
	})
	t.Run("CacheExpired", func(t *testing.T) {
		assert := assert.New(t)

		mr, n, err := newNodeWithMiniredis(t)
		if !assert.NoError(err, "Should create node with client") {
			t.FailNow()
		}

		mr.Close()
		n.roleCache.Save(master, "")
		n.roleCache.expire = time.Now().Add(-time.Minute)

		assert.Error(n.master(t.Context()), "Should not hit the cache and error out instead")
	})
	t.Run("CacheEmpty", func(t *testing.T) {
		assert := assert.New(t)

		mr, n, err := newNodeWithMiniredis(t)
		if !assert.NoError(err, "Should create node with client") {
			t.FailNow()
		}

		mr.Close()

		assert.Error(n.master(t.Context()), "Should not hit the cache and error out instead")
		assert.Empty(n.roleCache, "Should not save to cache on error")
	})
}

func TestNodeSlave(t *testing.T) {
	t.Run("NoClient", func(t *testing.T) {
		assert.NoError(t, (&node{}).slave(t.Context(), "testmaster"), "Should not panic when client is nil")
	})
	t.Run("CacheHit", func(t *testing.T) {
		assert := assert.New(t)

		mr, n, err := newNodeWithMiniredis(t)
		if !assert.NoError(err, "Should create node with client") {
			t.FailNow()
		}

		mr.Close()
		n.roleCache.Save(slave, "testmaster")

		assert.Nil(n.slave(t.Context(), "testmaster"), "Should hit cache and not attempt to connect to server")
	})
	t.Run("CacheExpired", func(t *testing.T) {
		assert := assert.New(t)

		mr, n, err := newNodeWithMiniredis(t)
		if !assert.NoError(err, "Should create node with client") {
			t.FailNow()
		}

		mr.Close()
		n.roleCache.Save(slave, "testmaster")
		n.roleCache.expire = time.Now().Add(-time.Minute)

		assert.Error(n.slave(t.Context(), "testmaster"), "Should not hit the cache and error out instead")
	})
	t.Run("CacheEmpty", func(t *testing.T) {
		assert := assert.New(t)

		mr, n, err := newNodeWithMiniredis(t)
		if !assert.NoError(err, "Should create node with client") {
			t.FailNow()
		}

		mr.Close()

		assert.Error(n.slave(t.Context(), "testmaster"), "Should not hit the cache and error out instead")
		assert.Empty(n.roleCache, "Should not save to cache on error")
	})
}

func TestNodeCacheSave(t *testing.T) {
	_, c := newSetupAndClient(t, "node-cache-save", 2)

	for i, n := range c.nodes {
		err := n.connect(t.Context(), valkey.ClientOption{})
		if !assert.NoErrorf(t, err, "Should connect to node %d", i) {
			t.FailNow()
		}
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

		err := n.slave(t.Context(), c.virtualAddress)

		assert.NoError(err, "Should set node to slave")
		assert.Equal(slave, n.roleCache.role, "Should save role in cache")
		assert.Equal(c.virtualAddress, n.roleCache.masterHost, "Should save master_host in cache")
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
