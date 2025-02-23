package failoverclient

import (
	"testing"

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
	err := n.connect(t.Context(), valkey.ClientOption{})

	assert.Error(err, "Should fail to retrieve information")
	assert.Nil(n.client, "Should not set client")
	assert.False(n.up, "Should not mark node as up")
	assert.Empty(n.runID, "Should have no run_id")
}

func TestNodePing(t *testing.T) {
	assert := assert.New(t)
	mr := miniredis.RunT(t)
	ctx := t.Context()

	opt := valkey.ClientOption{
		InitAddress:  []string{mr.Addr()},
		DisableCache: true,
		DisableRetry: true,
	}
	client, err := valkey.NewClient(opt)
	if !assert.NoError(err, "Should create valkey client") {
		t.FailNow()
	}

	n := &node{
		address: mr.Host(),
		port:    int64(mr.Server().Addr().Port),
		client:  client,
	}

	n.ping(ctx)
	assert.NotNil(n.client, "Node should retain client")
	assert.True(n.up, "Node should be up")

	mr.Close()

	n.ping(ctx)
	assert.Nil(n.client, "Node should close client")
	assert.False(n.up, "Node should be down")
}
