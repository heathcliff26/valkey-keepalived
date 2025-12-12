package utils

import (
	"fmt"
	"os"
	"strconv"
	"sync"
)

const haproxyConfigTemplate = `global
	log stdout format raw local0 info

	defaults
		timeout client 1s
		timeout connect 1s
		timeout server 1s

	frontend valkey-failover
		mode tcp
		bind :%d
		default_backend valkey

	backend valkey
		balance first
%s`

var setupLock sync.Mutex

type TestNode struct {
	Name string
	Port int
}

type FailoverSetup struct {
	Prefix            string
	Address           string
	Port              int
	Nodes             []TestNode
	runningNodes      []bool
	haproxyConfigPath string
	runningHAProxy    bool
}

// Create a new test setup, spawning containers.
// To ensure no port/image pull collisions occur, this function can only run one at a time through a global lock.
func NewFailoverSetup(prefix string, nodes int) (*FailoverSetup, error) {
	if prefix == "" {
		return nil, fmt.Errorf("need to provide a prefix")
	}
	if nodes < 2 {
		return nil, fmt.Errorf("need at least 2 nodes")
	}

	setupLock.Lock()
	defer setupLock.Unlock()

	res := &FailoverSetup{
		Prefix:       prefix,
		Nodes:        make([]TestNode, nodes),
		runningNodes: make([]bool, nodes),
	}

	var haproxyCfgBackend string

	for i := range nodes {
		res.Nodes[i].Name = fmt.Sprintf("%s-valkey-%d", prefix, i)
		port, err := findFreePort()
		if err != nil {
			res.Cleanup()
			return nil, err
		}
		err = ExecCRI("run", "-d", "--name", res.Nodes[i].Name, "--net", "host", "docker.io/valkey/valkey:latest", "valkey-server", "--port", strconv.Itoa(port))
		if err != nil {
			res.Cleanup()
			return nil, err
		}
		res.runningNodes[i] = true
		res.Nodes[i].Port = port

		haproxyCfgBackend += fmt.Sprintf("		server valkey-%s localhost:%d check\n", res.Nodes[i].Name, res.Nodes[i].Port)
	}

	port, err := findFreePort()
	if err != nil {
		res.Cleanup()
		return nil, err
	}
	haproxyCfg := fmt.Sprintf(haproxyConfigTemplate, port, haproxyCfgBackend)

	file, err := os.CreateTemp("", fmt.Sprintf("%s-*-haproxy.cfg", prefix))
	if err != nil {
		res.Cleanup()
		return nil, err
	}
	defer file.Close()

	res.haproxyConfigPath = file.Name()

	_, err = file.WriteString(haproxyCfg)
	if err != nil {
		res.Cleanup()
		return nil, err
	}
	err = file.Chmod(0644)
	if err != nil {
		res.Cleanup()
		return nil, err
	}

	err = ExecCRI("run", "-d", "-v", fmt.Sprintf("%s:/usr/local/etc/haproxy/haproxy.cfg:z", file.Name()), "--name", fmt.Sprintf("%s-haproxy", prefix), "--net", "host", "docker.io/library/haproxy:alpine")
	if err != nil {
		res.Cleanup()
		return nil, err
	}
	res.runningHAProxy = true
	res.Port = port
	res.Address = "localhost"

	return res, nil
}

func (s *FailoverSetup) StopNode(i int) error {
	if i >= len(s.Nodes) {
		return fmt.Errorf("index is out of range, max %d but got %d", len(s.Nodes)-1, i)
	}
	if !s.runningNodes[i] {
		return fmt.Errorf("node is already stopped")
	}

	err := ExecCRI("stop", s.Nodes[i].Name)

	s.runningNodes[i] = err != nil
	return err
}

func (s *FailoverSetup) StartNode(i int) error {
	if i >= len(s.Nodes) {
		return fmt.Errorf("index is out of range, max %d but got %d", len(s.Nodes)-1, i)
	}
	if s.runningNodes[i] {
		return fmt.Errorf("node is already running")
	}

	err := ExecCRI("start", s.Nodes[i].Name)

	s.runningNodes[i] = err == nil
	return err
}

func (s *FailoverSetup) StopHAProxy() error {
	if !s.runningHAProxy {
		return fmt.Errorf("haproxy is already stopped")
	}

	err := ExecCRI("stop", fmt.Sprintf("%s-haproxy", s.Prefix))

	s.runningHAProxy = err != nil
	return err
}

func (s *FailoverSetup) StartHAProxy() error {
	if s.runningHAProxy {
		return fmt.Errorf("haproxy is already stopped")
	}

	err := ExecCRI("start", fmt.Sprintf("%s-haproxy", s.Prefix))

	s.runningHAProxy = err == nil
	return err
}

func (s *FailoverSetup) Cleanup() {
	fmt.Printf("Cleaning up after test %s\n", s.Prefix)

	if s.runningHAProxy {
		err := s.StopHAProxy()
		if err != nil {
			fmt.Printf("Failed to stop haproxy: %v\n", err)
		}
	}
	err := ExecCRI("rm", fmt.Sprintf("%s-haproxy", s.Prefix))
	if err != nil {
		fmt.Printf("Failed to remove haproxy container: %v\n", err)
	}
	if s.haproxyConfigPath != "" {
		err := os.Remove(s.haproxyConfigPath)
		if err != nil {
			fmt.Printf("Failed to remove haproxy config file at \"%s\": %v\n", s.haproxyConfigPath, err)
		}
	}

	for i, node := range s.Nodes {
		if s.runningNodes[i] {
			err := s.StopNode(i)
			if err != nil {
				fmt.Printf("Failed to stop node %s: %v\n", node.Name, err)
			}
		}
		err := ExecCRI("rm", node.Name)
		if err != nil {
			fmt.Printf("Failed to remove node %s: %v\n", node.Name, err)
		}
	}

	fmt.Printf("Finished cleanup for test %s\n", s.Prefix)
}
