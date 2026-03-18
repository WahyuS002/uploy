package handlers

import (
	"net/http"
	"os/exec"
)

func DockerPsHandler(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("docker", "ps").Output()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Write(out)
}
