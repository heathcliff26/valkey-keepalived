package failoverclient

import (
	"context"
	"log/slog"

	"github.com/valkey-io/valkey-go"
)

type node struct {
	address string
	port    int64
	runID   string
	up      bool
	client  valkey.Client
}

const failedToConnectToNodeMsg = "Failed to connect to node"

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

	n.runID = parseRunIDFromInfo(res)
	n.client = client
	n.up = true

	return nil
}

func (n *node) ping(ctx context.Context) {
	res, err := n.client.Do(ctx, n.client.B().Ping().Build()).ToString()
	if err != nil || res != "PONG" {
		if n.up {
			slog.Info("Node is DOWN", slog.String("node", n.address), "err", err, slog.String("res", res))
			n.up = false
		}
		n.client.Close()
		n.client = nil
	} else if !n.up {
		n.up = true
		slog.Info("Node is UP", slog.String("node", n.address))
	}
}

func (n *node) master(ctx context.Context) error {
	if n.client == nil {
		slog.Debug("Node is not up, skipping for update", slog.String("node", n.address))
		return nil
	}

	return n.client.Do(ctx, n.client.B().Replicaof().No().One().Build()).Error()
}

func (n *node) slave(ctx context.Context, master string) error {
	if n.client == nil {
		slog.Debug("Node is not up, skipping for update", slog.String("node", n.address))
		return nil
	}

	return n.client.Do(ctx, n.client.B().Replicaof().Host(master).Port(n.port).Build()).Error()
}

func (n *node) close() {
	if n.client != nil {
		n.client.Close()
		n.client = nil
	}
}
