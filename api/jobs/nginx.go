package jobs

import "os/exec"

func RunNginx() {
	exec.Command("docker", "pull", "nginx:latest").Run()
}
