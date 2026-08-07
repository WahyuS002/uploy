package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/WahyuS002/uploy/auth"
	"github.com/WahyuS002/uploy/db"
	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/respond"
	"github.com/WahyuS002/uploy/ssh"
	"github.com/jackc/pgx/v5"
)

// How much history Docker replays before the stream catches up to live. Enough
// to show why a container is unhappy without pushing a wall of text at someone
// who only wanted to watch what happens next.
const logTailLines = 200

// GetServiceLogs streams a running container's stdout and stderr over SSE.
//
// Unlike deployment logs there is nothing to persist or replay: Docker already
// keeps the history, --tail covers the backfill, and a reconnect just re-tails.
// That also means no broker — a deployment fans out to everyone watching it,
// while this is one SSH session per viewer.
func (s *Server) GetServiceLogs(w http.ResponseWriter, r *http.Request, id string) {
	sc, _ := auth.GetSessionContext(r)

	flusher, ok := w.(http.Flusher)
	if !ok {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "streaming not supported"})
		return
	}

	svc, err := db.GetServiceWithServer(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "service not found"})
		} else {
			log.Printf("GetServiceWithServer id=%s error: %v", id, err)
			respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "failed to look up service"})
		}
		return
	}

	// Same as everywhere else: 404 rather than 403 across workspaces, so the
	// response does not confirm that the service exists.
	if svc.WorkspaceID != sc.WorkspaceID {
		respond.JSON(w, http.StatusNotFound, gen.ErrorResponse{Error: "service not found"})
		return
	}

	if !svc.HasDeployed {
		respond.JSON(w, http.StatusBadRequest, gen.ErrorResponse{Error: "service has not been deployed yet, so it has no logs"})
		return
	}

	// Connect before switching the response to SSE: once the event-stream headers
	// are out, a failure can only be reported as an in-band event, and an
	// unreachable server deserves a real status code.
	//
	// ponytail: one SSH connection per viewer, opened on demand. Pool them if
	// many people end up watching logs on the same server at once.
	client, err := ssh.NewClient(ssh.ServerConfig{
		Host:       svc.Host,
		Port:       int(svc.ServerPort),
		User:       svc.SSHUser,
		PrivateKey: svc.PrivateKey,
	})
	if err != nil {
		log.Printf("service logs id=%s: ssh connect: %v", id, err)
		respond.JSON(w, http.StatusBadGateway, gen.ErrorResponse{Error: "could not connect to the server: " + err.Error()})
		return
	}
	defer client.Close()

	if err := client.DetectDocker(); err != nil {
		log.Printf("service logs id=%s: detect docker: %v", id, err)
		respond.JSON(w, http.StatusBadGateway, gen.ErrorResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	sendLine := func(output, logType string) {
		data, err := json.Marshal(map[string]string{"output": output, "type": logType})
		if err != nil {
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	sendError := func(msg string) {
		payload, err := json.Marshal(map[string]string{"message": msg})
		if err != nil {
			return
		}
		fmt.Fprintf(w, "event: stream-error\ndata: %s\n\n", payload)
		flusher.Flush()
	}

	// Container names are validated on the way in (validContainerName), so
	// interpolating one here cannot inject anything.
	cmd := fmt.Sprintf("%s logs -f --tail %d %s", client.DockerBin(), logTailLines, svc.ContainerName)
	stdoutCh, stderrCh, done := client.StreamCommandContext(r.Context(), cmd)

	// Only this goroutine writes to w. Nil out each channel as it closes so the
	// select stops waiting on a drained side.
	for stdoutCh != nil || stderrCh != nil {
		select {
		case <-r.Context().Done():
			return
		case line, ok := <-stdoutCh:
			if !ok {
				stdoutCh = nil
				continue
			}
			sendLine(line, "stdout")
		case line, ok := <-stderrCh:
			if !ok {
				stderrCh = nil
				continue
			}
			// Docker sends the container's own stderr here; it is ordinary
			// application output, not a failure of the stream.
			sendLine(line, "stderr")
		}
	}

	// The stream ended on its own: the container stopped, or docker logs failed.
	// A client that has already disconnected is not an error worth reporting.
	if err := <-done; err != nil && r.Context().Err() == nil {
		sendError(err.Error())
		return
	}

	fmt.Fprint(w, "event: done\ndata: stream ended\n\n")
	flusher.Flush()
}
