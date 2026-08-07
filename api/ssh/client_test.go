package ssh

import (
	"context"
	"errors"
	"net"
	"strconv"
	"testing"
	"time"

	gossh "golang.org/x/crypto/ssh"
)

// startFollowServer runs an in-process SSH server whose only command writes one
// line and then never exits, the way "docker logs -f" behaves. It is the only
// way to exercise the cancellation path: the failure it guards against is a
// stream that keeps running after the caller is gone, which no amount of
// checking a returned value would catch.
//
// Returns the address to dial and a client private key the server accepts.
func startFollowServer(t *testing.T) (addr string, clientKeyPEM string) {
	t.Helper()

	hostKeyPEM, err := GenerateEd25519Key()
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	hostSigner, err := gossh.ParsePrivateKey([]byte(hostKeyPEM))
	if err != nil {
		t.Fatalf("parse host key: %v", err)
	}

	clientKeyPEM, err = GenerateEd25519Key()
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

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // listener closed by cleanup
			}
			go serveFollowConn(conn, config)
		}
	}()

	return listener.Addr().String(), clientKeyPEM
}

func serveFollowConn(conn net.Conn, config *gossh.ServerConfig) {
	defer conn.Close()

	sshConn, chans, reqs, err := gossh.NewServerConn(conn, config)
	if err != nil {
		return
	}
	defer sshConn.Close()
	go gossh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(gossh.UnknownChannelType, "only sessions")
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			return
		}

		go func() {
			for req := range requests {
				if req.Type != "exec" {
					req.Reply(false, nil)
					continue
				}
				req.Reply(true, nil)
				channel.Write([]byte("first line\n"))
				// Then hang, like a follow command does. The only thing that ends
				// this is the client tearing the session down.
			}
		}()
	}
}

// A cancelled context has to actually stop the stream. Without it every viewer
// who navigates away from a log tab strands a goroutine, an SSH session, and a
// remote process — a leak that only shows up under load, long after the change
// that caused it.
func TestStreamCommandContextCancelStopsStream(t *testing.T) {
	addr, clientKey := startFollowServer(t)

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split addr: %v", err)
	}
	portNum, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}

	client, err := NewClient(ServerConfig{
		Host:       host,
		Port:       portNum,
		User:       "tester",
		PrivateKey: clientKey,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	stdout, _, done := client.StreamCommandContext(ctx, "docker logs -f whatever")

	select {
	case line, ok := <-stdout:
		if !ok {
			t.Fatal("stdout closed before any output")
		}
		if line != "first line" {
			t.Fatalf("got %q, want %q", line, "first line")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the first line")
	}

	// The command is still running and nobody is reading stdout any more, which
	// is exactly the state a disconnected viewer leaves behind.
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("done = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("stream did not stop after the context was cancelled")
	}

	select {
	case _, ok := <-stdout:
		if ok {
			t.Error("stdout still open after cancellation")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("stdout was never closed after cancellation")
	}
}
