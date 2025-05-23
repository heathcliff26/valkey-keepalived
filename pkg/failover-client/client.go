package failoverclient

import (
	"context"
	"crypto/tls"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/valkey-io/valkey-go"
)

type FailoverClient struct {
	clientOption   valkey.ClientOption
	nodes          []*node
	virtualAddress string
	port           int64
	currentMaster  string
	masterAddr     string

	quit chan os.Signal
}

// Create a new failover client from the given configuration
func NewFailoverClient(cfg ValkeyConfig) *FailoverClient {
	option := valkey.ClientOption{
		Username:     cfg.Username,
		Password:     cfg.Password,
		DisableCache: true,
		DisableRetry: true,
	}
	if cfg.TLS {
		option.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	nodes := make([]*node, len(cfg.Nodes))

	for i, addr := range cfg.Nodes {
		nodes[i] = &node{
			address:   addr,
			port:      cfg.Port,
			up:        true,
			roleCache: &roleCache{},
		}
	}

	return &FailoverClient{
		clientOption:   option,
		nodes:          nodes,
		virtualAddress: cfg.VirtualAddress,
		port:           cfg.Port,
		quit:           make(chan os.Signal, 1),
	}
}

// Run the given function on all nodes in parallel and wait
func (c *FailoverClient) parallelJob(timeout time.Duration, f func(context.Context, *node)) {
	var wg sync.WaitGroup

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for _, n := range c.nodes {
		wg.Add(1)

		go func() {
			defer wg.Done()

			f(ctx, n)
		}()
	}

	wg.Wait()
}

// Check the current status of all nodes
func (c *FailoverClient) updateNodes() {
	c.parallelJob(time.Second, func(ctx context.Context, n *node) {
		if n.client == nil {
			err := n.connect(ctx, c.clientOption)
			if err != nil {
				logLevel := slog.LevelDebug
				if n.up {
					logLevel = slog.LevelWarn
					n.up = false
				}
				slog.Log(ctx, logLevel, nodeDownMsg, slog.String("node", n.address), "err", err)
			}
			return
		}

		n.ping(ctx)
	})
}

// Continuosly check the current status and failover if necessary
func (c *FailoverClient) Run() {
	signal.Notify(c.quit, os.Interrupt, syscall.SIGTERM)

	firstTime := true

	slog.Info("Starting failover client")
	for {
		if !firstTime {
			select {
			case <-c.quit:
				slog.Info("Shutting down failover client")
				return
			case <-time.After(time.Second):
			}
		} else {
			firstTime = false
		}

		c.updateNodes()

		client, err := newValkeyClient(c.virtualAddress, c.port, c.clientOption)
		if err != nil {
			slog.Error("Failed to connect to virtual address", slog.String("addr", c.virtualAddress), "err", err)
			continue
		}

		res, err := client.Do(context.Background(), client.B().Info().Section("server").Build()).ToString()
		client.Close()
		if err != nil {
			slog.Error("Failed to retrieve info from virtual address", slog.String("addr", c.virtualAddress), "err", err)
			continue
		}
		currentMaster := ParseValueFromInfo(res, runID)
		if currentMaster != c.currentMaster {
			found := false
			for _, n := range c.nodes {
				if n.runID == currentMaster {
					c.currentMaster = currentMaster
					c.masterAddr = n.address
					found = true
				}
			}
			if !found {
				slog.Error("Could not find the current masters addr", slog.String(runID, currentMaster))
				continue
			} else {
				slog.Info("Switching over to new master", slog.String("addr", c.masterAddr), slog.String(runID, c.currentMaster))
			}
		}

		c.parallelJob(time.Second, func(ctx context.Context, n *node) {
			if n.runID == c.currentMaster {
				err := n.master(ctx)
				if err != nil {
					slog.Error("Failed to update node to master", slog.String("node", n.address), "err", err)
				}
			} else {
				err := n.slave(ctx, c.masterAddr)
				if err != nil {
					slog.Error("Failed to update node to slave", slog.String("node", n.address), "err", err)
				}
			}
		})
	}
}

// Close all open client connections
func (c *FailoverClient) Close() {
	for _, n := range c.nodes {
		n.close()
	}
}
