package utils

import "net"

// Find a free TCP port on the host machine
func findFreePort() (int, error) {
	ln, err := net.Listen("tcp", "")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	addr := ln.Addr().(*net.TCPAddr)
	return addr.Port, nil
}
