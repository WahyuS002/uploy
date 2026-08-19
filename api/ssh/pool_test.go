package ssh

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gossh "golang.org/x/crypto/ssh"
)

// tunnelServer is an in-process SSH server that answers global requests and
// forwards direct-tcpip channels, which is everything a tunneled HTTP call
// touches. It counts handshakes so a test can tell a reused connection from a
// fresh one, and can sever live connections to imitate a server that went away.
type tunnelServer struct {
	addr      string
	clientKey string

	handshakes atomic.Int32

	mu   sync.Mutex
	live []net.Conn
}

func startTunnelServer(t *testing.T) *tunnelServer {
	t.Helper()

	hostKeyPEM, err := GenerateEd25519Key()
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	hostSigner, err := gossh.ParsePrivateKey([]byte(hostKeyPEM))
	if err != nil {
		t.Fatalf("parse host key: %v", err)
	}
	clientKeyPEM, err := GenerateEd25519Key()
	if err != nil {
		t.Fatalf("generate client key: %v", err)
	}

	config := &gossh.ServerConfig{
		PublicKeyCallback: func(gossh.ConnMetadata, gossh.PublicKey) (*gossh.Permissions, error) {
			return nil, nil // the test's own key is the only one that can reach this
		},
	}
	config.AddHostKey(hostSigner)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { listener.Close() })

	server := &tunnelServer{addr: listener.Addr().String(), clientKey: clientKeyPEM}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // listener closed by cleanup
			}
			server.track(conn)
			go server.serve(conn, config)
		}
	}()
	t.Cleanup(server.disconnect)
	return server
}

func (s *tunnelServer) config() ServerConfig {
	host, port, _ := net.SplitHostPort(s.addr)
	number, _ := strconv.Atoi(port)
	return ServerConfig{Host: host, Port: number, User: "tester", PrivateKey: s.clientKey}
}

func (s *tunnelServer) track(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.live = append(s.live, conn)
}

// disconnect drops every live connection, the way a reboot or a network blip
// would. Pooled clients only find out on their next request.
func (s *tunnelServer) disconnect() {
	s.mu.Lock()
	live := s.live
	s.live = nil
	s.mu.Unlock()
	for _, conn := range live {
		conn.Close()
	}
}

func (s *tunnelServer) serve(conn net.Conn, config *gossh.ServerConfig) {
	defer conn.Close()

	sshConn, chans, reqs, err := gossh.NewServerConn(conn, config)
	if err != nil {
		return
	}
	s.handshakes.Add(1)
	defer sshConn.Close()
	go gossh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "direct-tcpip" {
			newChannel.Reject(gossh.UnknownChannelType, "only direct-tcpip")
			continue
		}
		go forwardChannel(newChannel)
	}
}

// forwardChannel dials the address the client asked for and pipes the channel to
// it, which is what an SSH server does for a port-forwarded connection.
func forwardChannel(newChannel gossh.NewChannel) {
	target, err := parseDirectTCPIP(newChannel.ExtraData())
	if err != nil {
		newChannel.Reject(gossh.ConnectionFailed, "bad payload")
		return
	}
	upstream, err := net.Dial("tcp", target)
	if err != nil {
		newChannel.Reject(gossh.ConnectionFailed, "dial failed")
		return
	}
	channel, requests, err := newChannel.Accept()
	if err != nil {
		upstream.Close()
		return
	}
	go gossh.DiscardRequests(requests)
	go func() {
		defer channel.Close()
		defer upstream.Close()
		io.Copy(upstream, channel)
	}()
	go func() {
		defer channel.Close()
		defer upstream.Close()
		io.Copy(channel, upstream)
	}()
}

func parseDirectTCPIP(payload []byte) (string, error) {
	host, rest, err := readSSHString(payload)
	if err != nil {
		return "", err
	}
	if len(rest) < 4 {
		return "", io.ErrUnexpectedEOF
	}
	port := binary.BigEndian.Uint32(rest[:4])
	return net.JoinHostPort(host, strconv.Itoa(int(port))), nil
}

func readSSHString(payload []byte) (string, []byte, error) {
	if len(payload) < 4 {
		return "", nil, io.ErrUnexpectedEOF
	}
	length := binary.BigEndian.Uint32(payload[:4])
	if uint32(len(payload)-4) < length {
		return "", nil, io.ErrUnexpectedEOF
	}
	return string(payload[4 : 4+length]), payload[4+length:], nil
}

// Reuse is the whole point of the pool: an open dashboard polls, and a handshake
// per poll would cost more than the query it carries.
func TestPoolReusesOneConnection(t *testing.T) {
	server := startTunnelServer(t)
	pool := NewPool()
	defer pool.Close()

	for range 3 {
		if _, err := Use(pool, server.config(), func(client *Client) (bool, error) {
			return client.alive(), nil
		}); err != nil {
			t.Fatalf("Use: %v", err)
		}
	}

	if got := server.handshakes.Load(); got != 1 {
		t.Fatalf("handshakes = %d; want 1 connection reused across calls", got)
	}
}

// A pooled connection can be severed between requests, and the break only
// surfaces on the next use. Reporting that as a failure would turn every server
// reboot into a broken dashboard until something else happened to reconnect.
func TestPoolReconnectsAfterConnectionDies(t *testing.T) {
	server := startTunnelServer(t)
	pool := NewPool()
	defer pool.Close()

	if _, err := Use(pool, server.config(), func(*Client) (bool, error) { return true, nil }); err != nil {
		t.Fatalf("first Use: %v", err)
	}
	server.disconnect()
	waitForDeadConnection(t, pool, server.config())

	attempts := 0
	alive, err := Use(pool, server.config(), func(client *Client) (bool, error) {
		attempts++
		if !client.alive() {
			return false, io.ErrUnexpectedEOF
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("Use after disconnect: %v", err)
	}
	if !alive {
		t.Fatal("expected the retry to run on a live connection")
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d; want the dead connection dropped and the call retried once", attempts)
	}
	if got := server.handshakes.Load(); got != 2 {
		t.Fatalf("handshakes = %d; want a second connection dialed", got)
	}
}

// A connection dialed for this very call is not worth retrying: the failure came
// from the request, not from a connection that went stale in the pool.
func TestPoolDoesNotRetryFreshConnection(t *testing.T) {
	server := startTunnelServer(t)
	pool := NewPool()
	defer pool.Close()

	attempts := 0
	_, err := Use(pool, server.config(), func(*Client) (bool, error) {
		attempts++
		return false, io.ErrUnexpectedEOF
	})
	if err == nil {
		t.Fatal("expected the call error to be reported")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d; want 1", attempts)
	}
}

// An error from a healthy connection is the agent's, not the tunnel's, and must
// reach the caller untouched rather than being retried.
func TestPoolReportsCallErrorOnLiveConnection(t *testing.T) {
	server := startTunnelServer(t)
	pool := NewPool()
	defer pool.Close()

	if _, err := Use(pool, server.config(), func(*Client) (bool, error) { return true, nil }); err != nil {
		t.Fatalf("warm-up Use: %v", err)
	}

	attempts := 0
	_, err := Use(pool, server.config(), func(*Client) (bool, error) {
		attempts++
		return false, io.ErrUnexpectedEOF
	})
	if err == nil {
		t.Fatal("expected the call error to be reported")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d; want the error passed through without a retry", attempts)
	}
}

func TestPoolEvictsIdleConnections(t *testing.T) {
	server := startTunnelServer(t)
	pool := NewPool()
	defer pool.Close()

	if _, err := Use(pool, server.config(), func(*Client) (bool, error) { return true, nil }); err != nil {
		t.Fatalf("Use: %v", err)
	}

	pool.evictIdle(time.Now().Add(poolIdleTTL + time.Minute))

	pool.mu.Lock()
	remaining := len(pool.entries)
	pool.mu.Unlock()
	if remaining != 0 {
		t.Fatalf("entries = %d; want the idle connection dropped", remaining)
	}
}

// A connection in use must survive the sweeper, or a long-running read would
// have the ground pulled out from under it.
func TestPoolKeepsBorrowedConnectionThroughSweep(t *testing.T) {
	server := startTunnelServer(t)
	pool := NewPool()
	defer pool.Close()

	_, err := Use(pool, server.config(), func(client *Client) (bool, error) {
		pool.evictIdle(time.Now().Add(poolIdleTTL + time.Minute))
		pool.mu.Lock()
		remaining := len(pool.entries)
		pool.mu.Unlock()
		if remaining != 1 {
			t.Errorf("entries = %d; want the borrowed connection kept", remaining)
		}
		return client.alive(), nil
	})
	if err != nil {
		t.Fatalf("Use: %v", err)
	}
}

// Rotating a server's key must not hand back a connection authenticated with the
// old one.
func TestPoolKeySeparatesCredentials(t *testing.T) {
	base := ServerConfig{Host: "10.0.0.4", Port: 22, User: "deploy", PrivateKey: "key-one"}
	rotated := base
	rotated.PrivateKey = "key-two"
	otherUser := base
	otherUser.User = "root"

	if poolKey(base) == poolKey(rotated) {
		t.Fatal("rotated key must not share a pool entry")
	}
	if poolKey(base) == poolKey(otherUser) {
		t.Fatal("different users must not share a pool entry")
	}
	if poolKey(base) != poolKey(base) {
		t.Fatal("the same credentials must map to one entry")
	}
}

func TestPoolClosedRejectsUse(t *testing.T) {
	server := startTunnelServer(t)
	pool := NewPool()
	pool.Close()

	if _, err := Use(pool, server.config(), func(*Client) (bool, error) { return true, nil }); err != ErrPoolClosed {
		t.Fatalf("err = %v; want ErrPoolClosed", err)
	}
}

// waitForDeadConnection blocks until the pooled connection notices it was cut.
// The client learns of a severed transport asynchronously, so asserting on the
// retry immediately after disconnect would be racy.
func waitForDeadConnection(t *testing.T, pool *Pool, cfg ServerConfig) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		pool.mu.Lock()
		entry, ok := pool.entries[poolKey(cfg)]
		pool.mu.Unlock()
		if !ok {
			t.Fatal("pooled connection disappeared before the retry could be observed")
		}
		if !entry.client.alive() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("pooled connection never registered as dead")
}

// The tunnel exists to carry HTTP to an agent listening on the server's
// loopback. Proving a channel opens is not the same as proving a request and
// response survive it, so this drives a real HTTP exchange end to end.
func TestDialCarriesHTTPToLoopbackService(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	server := startTunnelServer(t)
	client, err := NewClient(server.config())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	address := strings.TrimPrefix(upstream.URL, "http://")
	httpClient := &http.Client{Transport: &http.Transport{
		DisableKeepAlives: true,
		DialContext: func(context.Context, string, string) (net.Conn, error) {
			return client.Dial("tcp", address)
		},
	}}
	req, err := http.NewRequest(http.MethodGet, upstream.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer secret")

	response, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("tunneled request: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d; want 200", response.StatusCode)
	}
	body, _ := io.ReadAll(response.Body)
	if string(body) != `{"ok":true}` {
		t.Fatalf("body = %q", body)
	}
}
