package utils

import (
	"fmt"
	"os/exec"
	"strings"
)

var (
	containerRuntime string
	initialized      = false
)

func findContainerRuntime() string {
	for _, cmd := range []string{"docker", "podman"} {
		path, err := exec.LookPath(cmd)
		if err != nil {
			continue
		}
		// #nosec G204: Path of container runtime is determined dynamically and based of PATH.
		err = exec.Command(path, "ps").Run()
		if err == nil {
			fmt.Printf("Found container runtime %s, path=%s\n", cmd, path)
			return path
		}
	}
	fmt.Println("Did not find any container runtimes")
	return ""
}

func HasContainerRuntimer() bool {
	if !initialized {
		containerRuntime = findContainerRuntime()
		initialized = true
	}
	return containerRuntime != ""
}

func GetContainerIP(name string) (string, error) {
	cmd := GetCommand("container", "inspect", "-f", "'{{.NetworkSettings.IPAddress}}'", name)
	buf, err := cmd.Output()

	res, _ := strings.CutSuffix(string(buf), "\n")
	res = strings.ReplaceAll(res, "'", "")
	return res, err
}

func GetCommand(args ...string) *exec.Cmd {
	if strings.Contains(containerRuntime, "podman") {
		args = append([]string{containerRuntime}, args...)
		return exec.Command("sudo", args...)
	}
	// #nosec G204: Path of container runtime is determined dynamically and based of PATH.
	return exec.Command(containerRuntime, args...)
}

func ExecCRI(args ...string) error {
	if !initialized {
		containerRuntime = findContainerRuntime()
		initialized = true
	}
	out, err := GetCommand(args...).CombinedOutput()
	if err != nil {
		argsStr := strings.Join(args, " ")
		fmt.Printf("Output from \"%s %s\":\n%s\n", containerRuntime, argsStr, string(out))
	}
	return err
}
