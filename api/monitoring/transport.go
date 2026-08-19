package monitoring

import (
	"context"
	"net"
	"net/http"
	"strconv"

	"github.com/WahyuS002/uploy/ssh"
)

// Transport carries requests to a monitoring agent's HTTP API.
//
// The agent publishes its port on the server's loopback only, so reaching it
// means tunneling through the SSH connection Uploy already holds. Naming that as
// an interface leaves room for an agent-initiated transport later — for servers
// Uploy cannot open a connection to at all — without disturbing any of the call
// sites in agent.go.
type Transport interface {
	// BaseURL is the agent's root URL, without a trailing slash.
	BaseURL() string
	// Do sends one request. The deadline comes from the request context.
	Do(req *http.Request) (*http.Response, error)
}

type sshTransport struct {
	http *http.Client
	base string
}

// OverSSH tunnels agent requests through an established SSH connection.
//
// The address is resolved on the far side of the tunnel, so 127.0.0.1 here means
// the server's loopback rather than the control plane's. Keep-alives are off
// because each call opens its own tunnel: an idle pooled connection would hold
// an SSH channel open with nothing to say on it.
func OverSSH(client *ssh.Client, port int) Transport {
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	return &sshTransport{
		base: "http://" + address,
		http: &http.Client{
			Transport: &http.Transport{
				DisableKeepAlives: true,
				DialContext: func(context.Context, string, string) (net.Conn, error) {
					return client.Dial("tcp", address)
				},
			},
		},
	}
}

func (t *sshTransport) BaseURL() string { return t.base }

func (t *sshTransport) Do(req *http.Request) (*http.Response, error) {
	return t.http.Do(req)
}

// Target locates a monitoring agent: the SSH connection to reach it through, and
// the loopback port it listens on behind that.
type Target struct {
	SSH  ssh.ServerConfig
	Port int
}

// withTransport runs fn against target's agent over a pooled SSH connection.
//
// Pooling matters here rather than being an optimisation: an open dashboard
// polls, and paying a key exchange per poll would make the tunnel cost more than
// the query it carries.
func withTransport[T any](target Target, fn func(Transport) (T, error)) (T, error) {
	return ssh.Use(ssh.DefaultPool, target.SSH, func(client *ssh.Client) (T, error) {
		return fn(OverSSH(client, target.Port))
	})
}
