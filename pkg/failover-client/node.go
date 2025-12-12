package failoverclient

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/valkey-io/valkey-go"
)

const (
	master = "master"
	slave  = "slave"

	runID      = "run_id"
	role       = "role"
	masterHost = "master_host"
	masterPort = "master_port"
)

type node struct {
	address string
	port    int64
	runID   string
	up      bool
	client  valkey.Client

	// Caches the last successfully set role to reduce api calls
	roleCache *roleCache
}

const (
	nodeDownMsg = "Node is DOWN"
	nodeUpMsg   = "Node is UP"
)

// Connect to valkey and retrieve the run_id
func (n *node) connect(ctx context.Context, option valkey.ClientOption) error {
	client, err := newValkeyClient(n.address, n.port, option)
	if err != nil {
		return err
	}

	res, err := client.Do(ctx, client.B().Info().Section("server").Build()).ToString()
	if err != nil {
		client.Close()
		return err
	}

	n.runID = ParseValueFromInfo(res, runID)
	n.client = client
	n.up = true

	return nil
}

// Check that the node is up.
// Requires client to be present.
func (n *node) ping(ctx context.Context) {
	res, err := n.client.Do(ctx, n.client.B().Ping().Build()).ToString()
	if err != nil || res != "PONG" {
		if n.up {
			slog.Info(nodeDownMsg, slog.String("node", n.address), "err", err, slog.String("res", res))
			n.up = false
		}
		n.client.Close()
		n.client = nil
	} else if !n.up {
		n.up = true
		slog.Info(nodeUpMsg, slog.String("node", n.address))
	}
}

// Make this node a master node
func (n *node) master(ctx context.Context) error {
	if n.client == nil {
		return fmt.Errorf("node is not up")
	}
	if n.roleCache.IsMaster() {
		return nil
	}

	info, err := n.getReplicationInfo(ctx)
	if err != nil {
		return err
	}
	if ParseValueFromInfo(info, role) == master {
		n.roleCache.Save(master, nil)
		return nil
	}

	err = n.client.Do(ctx, n.client.B().Replicaof().No().One().Build()).Error()
	if err != nil {
		return err
	}
	n.roleCache.Save(master, nil)
	return nil
}

// Make this node a slave of the given master
func (n *node) slave(ctx context.Context, newMaster *node) error {
	if n.client == nil {
		slog.Debug("Node is not up, skipping for update", slog.String("node", n.address))
		return nil
	}

	if n.roleCache.IsSlaveOf(newMaster) {
		return nil
	}

	info, err := n.getReplicationInfo(ctx)
	if err != nil {
		return err
	}
	if infoSlaveOfNode(info, newMaster) {
		n.roleCache.Save(slave, newMaster)
		return nil
	}

	err = n.client.Do(ctx, n.client.B().Replicaof().Host(newMaster.address).Port(newMaster.port).Build()).Error()
	if err != nil {
		return err
	}
	n.roleCache.Save(slave, newMaster)
	return nil
}

// Fetch the replication information from valkey
func (n *node) getReplicationInfo(ctx context.Context) (string, error) {
	return n.client.Do(ctx, n.client.B().Info().Section("replication").Build()).ToString()
}

// Close the open client
func (n *node) close() {
	if n.client != nil {
		n.client.Close()
		n.client = nil
	}
}

type roleCache struct {
	role   string
	master *node

	expire time.Time
}

// Save the current role and masterHost to the cache.
// Resets the cache expire time
func (rc *roleCache) Save(role string, master *node) {
	rc.role = role
	rc.master = master
	rc.expire = time.Now().Add(time.Minute)
}

func (rc *roleCache) isExpired() bool {
	return time.Now().After(rc.expire)
}

// Check if the current cache is a master
func (rc *roleCache) IsMaster() bool {
	if rc.isExpired() {
		return false
	}
	return rc.role == master
}

// Check if the current cache is a slave of the given master_host
func (rc *roleCache) IsSlaveOf(master *node) bool {
	if rc.isExpired() || master == nil {
		return false
	}
	return rc.role == slave && rc.master != nil && rc.master.address == master.address && rc.master.port == master.port
}
