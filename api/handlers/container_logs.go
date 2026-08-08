package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/WahyuS002/uploy/gen"
	"github.com/WahyuS002/uploy/respond"
	"github.com/WahyuS002/uploy/ssh"
)

// How much history Docker replays before the stream catches up to live. Enough
// to show why a container is unhappy without pushing a wall of text at someone
// who only wanted to watch what happens next.
const logTailLines = 200

// What each `since` value means to Docker. Go durations have no day unit, so the
// long ranges are spelled in hours. Being a lookup rather than a format string
// is also what makes the value safe to interpolate into a shell command:
// nothing outside these five keys can reach it.
var logSinceDuration = map[string]string{
	"1h":  "1h",
	"6h":  "6h",
	"24h": "24h",
	"7d":  "168h",
	"30d": "720h",
}

// logSinceFlag turns the requested range into the flag that narrows Docker's
// replay, or an empty string for "as far back as --tail reaches". ok is false
// for a range nobody offers — the only validation the value gets, since the
// generated binder accepts any string for an enum.
//
// It takes *string rather than either generated enum: the two endpoints that
// offer this range get their own named type for the same five values.
func logSinceFlag(since *string) (string, bool) {
	if since == nil {
		return "", true
	}
	d, ok := logSinceDuration[*since]
	if !ok {
		return "", false
	}
	return " --since " + d, true
}

// streamContainerLogs follows one container over SSH and relays it as SSE.
//
// Everything before this call differs per resource — which record, whose
// workspace, whether it has been deployed. Everything after it is the same
// stream, whether the container is somebody's app or the proxy in front of it.
//
// Nothing here is persisted or replayed: Docker already keeps the history,
// --tail covers the backfill, and a reconnect just re-tails. That also means no
// broker — a deployment fans out to everyone watching it, while this is one SSH
// session per viewer.
//
// The caller must not have written a response yet; the first thing this does is
// try to connect, and an unreachable server deserves a real status code rather
// than an in-band event nobody can act on.
//
// ponytail: one SSH connection per viewer, opened on demand. Pool them if many
// people end up watching logs on the same server at once.
func streamContainerLogs(w http.ResponseWriter, r *http.Request, cfg ssh.ServerConfig, container, sinceFlag, logTag string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		respond.JSON(w, http.StatusInternalServerError, gen.ErrorResponse{Error: "streaming not supported"})
		return
	}

	client, err := ssh.NewClient(cfg)
	if err != nil {
		log.Printf("%s: ssh connect: %v", logTag, err)
		respond.JSON(w, http.StatusBadGateway, gen.ErrorResponse{Error: "could not connect to the server: " + err.Error()})
		return
	}
	defer client.Close()

	if err := client.DetectDocker(); err != nil {
		log.Printf("%s: detect docker: %v", logTag, err)
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

	// Container names never come from the request — an app's is validated on the
	// way in (validContainerName) and the proxy's is a constant — so
	// interpolating one here cannot inject anything.
	// --tail applies on top of --since: a range narrows the history, it does not
	// lift the cap on how much of it arrives at once.
	cmd := fmt.Sprintf("%s logs -f --tail %d%s %s", client.DockerBin(), logTailLines, sinceFlag, container)
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
			// output, not a failure of the stream. Traefik in particular writes
			// its whole own log there while the access log goes to stdout.
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
