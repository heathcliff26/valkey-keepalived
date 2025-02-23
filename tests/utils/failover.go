package utils

import (
	"fmt"
	"os"
)

const haproxyConfigTemplate = `global
	log stdout format raw local0 info

	defaults
		timeout client 1s
		timeout connect 1s
		timeout server 1s

	frontend valkey
		mode tcp
		bind :6379
		default_backend valkey

	backend valkey
		balance first
`

type FailoverSetup struct {
	Prefix            string
	Nodes             []string
	runningNodes      []bool
	haproxyConfigPath string
	runningHAProxy    bool
}

func NewFailoverSetup(prefix string, nodes int) (*FailoverSetup, string, []string, error) {
	if prefix == "" {
		return nil, "", nil, fmt.Errorf("need to provide a prefix")
	}
	if nodes < 2 {
		return nil, "", nil, fmt.Errorf("need at least 2 nodes")
	}

	res := &FailoverSetup{
		Prefix:       prefix,
		Nodes:        make([]string, nodes),
		runningNodes: make([]bool, nodes),
	}

	nodeIPs := make([]string, nodes)

	haproxyCfg := haproxyConfigTemplate

	for i := range nodes {
		res.Nodes[i] = fmt.Sprintf("%s-valkey-%d", prefix, i)
		err := ExecCRI("run", "-d", "--name", res.Nodes[i], "docker.io/valkey/valkey:latest")
		if err != nil {
			res.Cleanup()
			return nil, "", nil, err
		}
		res.runningNodes[i] = true

		nodeIPs[i], err = GetContainerIP(res.Nodes[i])
		if err != nil {
			res.Cleanup()
			return nil, "", nil, err
		}
		haproxyCfg += fmt.Sprintf("		server valkey-%s %s:6379 check\n", res.Nodes[i], nodeIPs[i])
	}

	file, err := os.CreateTemp("", fmt.Sprintf("%s-*-haproxy.cfg", prefix))
	if err != nil {
		res.Cleanup()
		return nil, "", nil, err
	}
	defer file.Close()

	res.haproxyConfigPath = file.Name()

	_, err = file.WriteString(haproxyCfg)
	if err != nil {
		res.Cleanup()
		return nil, "", nil, err
	}
	err = file.Chmod(0644)
	if err != nil {
		res.Cleanup()
		return nil, "", nil, err
	}

	err = ExecCRI("run", "-d", "-v", fmt.Sprintf("%s:/usr/local/etc/haproxy/haproxy.cfg:z", file.Name()), "--name", fmt.Sprintf("%s-haproxy", prefix), "docker.io/library/haproxy:alpine")
	if err != nil {
		res.Cleanup()
		return nil, "", nil, err
	}
	res.runningHAProxy = true

	haproxyIP, err := GetContainerIP(fmt.Sprintf("%s-haproxy", prefix))
	if err != nil {
		res.Cleanup()
		return nil, "", nil, err
	}

	return res, haproxyIP, nodeIPs, nil
}

func (s *FailoverSetup) StopNode(i int) error {
	if i >= len(s.Nodes) {
		return fmt.Errorf("index is out of range, max %d but got %d", len(s.Nodes)-1, i)
	}
	if !s.runningNodes[i] {
		return fmt.Errorf("node is already stopped")
	}

	err := ExecCRI("stop", s.Nodes[i])

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

	err := ExecCRI("start", s.Nodes[i])

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
				fmt.Printf("Failed to stop node %s: %v\n", node, err)
			}
		}
		err := ExecCRI("rm", node)
		if err != nil {
			fmt.Printf("Failed to remove node %s: %v\n", node, err)
		}
	}

	fmt.Printf("Finished cleanup for test %s\n", s.Prefix)
}
