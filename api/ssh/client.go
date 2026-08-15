package ssh

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	gossh "golang.org/x/crypto/ssh"
)

// ErrInvalidPrivateKey is returned when the stored key cannot be parsed at all,
// which is a different failure from a server refusing a well-formed key.
// Callers branch on it, so it is a sentinel rather than a bare fmt.Errorf.
var ErrInvalidPrivateKey = errors.New("private key is not valid")

type Client struct {
	client    *gossh.Client
	user      string
	dockerBin string // detected by DetectDocker: "docker" or "sudo -n docker"
}

// DockerBin returns the detected docker command prefix.
// Must call DetectDocker first; defaults to "docker" if not called.
func (c *Client) DockerBin() string {
	if c.dockerBin == "" {
		return "docker"
	}
	return c.dockerBin
}

// IsRoot reports whether the SSH user is root.
func (c *Client) IsRoot() bool {
	return c.user == "root"
}

// DetectDocker probes whether docker is accessible directly or via sudo -n.
// Returns an error if neither works.
func (c *Client) DetectDocker() error {
	// Try direct access first (root, docker group, rootless)
	if err := c.runProbe("docker info >/dev/null 2>&1"); err == nil {
		c.dockerBin = "docker"
		return nil
	}

	// Try sudo -n (non-interactive, fails fast if password required)
	if c.user != "root" {
		if err := c.runProbe("sudo -n docker info >/dev/null 2>&1"); err == nil {
			c.dockerBin = "sudo -n docker"
			return nil
		}
	}

	return fmt.Errorf("docker is not accessible for user %q: tried direct and sudo -n", c.user)
}

func (c *Client) runProbe(cmd string) error {
	session, err := c.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	return session.Run(cmd)
}

type ServerConfig struct {
	Host       string
	Port       int
	User       string
	PrivateKey string // Fill private key SSH, not path file

	// TODO: Add KnownHostsPath or pinned host key support for server host key verification.
	// TODO: Add DialTimeout so the connection does not hang indefinitely.
	// TODO: Consider supporting passphrase-protected private keys if needed later.
}

func NewClient(cfg ServerConfig) (*Client, error) {
	signer, err := gossh.ParsePrivateKey([]byte(cfg.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPrivateKey, err)
	}

	config := &gossh.ClientConfig{
		User:            cfg.User,
		Auth:            []gossh.AuthMethod{gossh.PublicKeys(signer)},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,

		// TODO: Replace InsecureIgnoreHostKey with known_hosts or pinned host key verification.
		// TODO: Consider restricting SSH algorithms to secure SupportedAlgorithms().
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	client, err := gossh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("Cannot connect to %s: %w", addr, err)
	}

	// TODO: Return nil explicitly for clarity.
	return &Client{client: client, user: cfg.User}, err
}

func (c *Client) Close() {
	// TODO: Consider returning an error from Close() so the caller can handle failures.
	c.client.Close()
}

func (c *Client) Dial(network, address string) (net.Conn, error) {
	return c.client.Dial(network, address)
}

func (c *Client) Run(ctx context.Context, command string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	type result struct {
		output []byte
		err    error
	}
	done := make(chan result, 1)
	go func() {
		output, runErr := session.CombinedOutput(command)
		done <- result{output: output, err: runErr}
	}()

	select {
	case res := <-done:
		return strings.TrimSpace(string(res.output)), res.err
	case <-ctx.Done():
		_ = session.Close()
		<-done
		return "", ctx.Err()
	}
}

// RunElevated runs command directly for root clients and through non-interactive
// sudo for other users.
func (c *Client) RunElevated(ctx context.Context, command string) (string, error) {
	if !c.IsRoot() {
		command = "sudo -n " + command
	}
	return c.Run(ctx, command)
}

// TestSession opens an SSH session and runs "echo ok" to verify the server
// can actually execute commands, not just accept the TCP+auth handshake.
// A 10-second deadline covers session open + command execution, so a stalled
// remote cannot hang the caller indefinitely.
func (c *Client) TestSession() error {
	type result struct{ err error }
	done := make(chan result, 1)

	go func() {
		session, err := c.client.NewSession()
		if err != nil {
			done <- result{fmt.Errorf("failed to open session: %w", err)}
			return
		}
		defer session.Close()

		if err := session.Run("echo ok"); err != nil {
			done <- result{fmt.Errorf("failed to run test command: %w", err)}
			return
		}
		done <- result{nil}
	}()

	select {
	case r := <-done:
		return r.err
	case <-time.After(10 * time.Second):
		return fmt.Errorf("SSH session test timed out after 10s")
	}
}

// StreamCommand runs a command and returns its output line by line via channel.
//
// It never gives up on the command, so it suits only commands that end by
// themselves. For anything that follows output indefinitely, use
// StreamCommandContext.
func (c *Client) StreamCommand(command string) (<-chan string, <-chan string, <-chan error) {
	return c.StreamCommandContext(context.Background(), command)
}

// StreamCommandContext is StreamCommand with a way out.
//
// Cancelling ctx closes the SSH session, which kills the remote process and
// EOFs the pipes, and unblocks any send that is waiting on a consumer who has
// stopped reading. Both halves are needed: "docker logs -f" outlives the
// request that asked for it, so without this every viewer who navigated away
// would strand a goroutine, an SSH session, and a remote process.
//
// On cancellation done carries ctx.Err(), so callers can tell "the caller left"
// apart from "the command failed".
func (c *Client) StreamCommandContext(ctx context.Context, command string) (<-chan string, <-chan string, <-chan error) {
	stdout := make(chan string)
	stderr := make(chan string)
	done := make(chan error, 1)

	// TODO: Consider using buffered channels so reader goroutines do not block too easily when consumers are slow.
	// TODO: Consider merging stdout/stderr into a single event struct if stream ordering becomes important later.

	go func() {
		session, err := c.client.NewSession()
		if err != nil {
			close(stdout)
			close(stderr)
			done <- err
			return
		}

		outPipe, _ := session.StdoutPipe()
		errPipe, _ := session.StderrPipe()

		// TODO: Do not ignore the error from StdoutPipe().
		// TODO: Do not ignore the error from StderrPipe().
		// TODO: If either pipe creation fails, close the session and return the error.

		if err := session.Start(command); err != nil {
			session.Close()
			close(stdout)
			close(stderr)
			done <- err
			return
		}

		// Closing the session is what actually stops a running command: it tears
		// down the remote process and EOFs the pipes, so the scanners below fall
		// out of their loops and Wait returns.
		finished := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				session.Close()
			case <-finished:
			}
		}()

		// A send that no one is reading has to lose to cancellation, otherwise
		// the reader goroutine parks on it forever and wg.Wait never returns.
		send := func(ch chan<- string, line string) bool {
			select {
			case ch <- line:
				return true
			case <-ctx.Done():
				return false
			}
		}

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(outPipe)

			// TODO: Increase the scanner buffer if log output may contain long lines (> 64KB).
			// TODO: Consider using bufio.Reader instead if long-line handling needs to be safer later.

			for scanner.Scan() {
				if !send(stdout, scanner.Text()) {
					return
				}
			}

			// TODO: Check scanner.Err() so stdout read errors are not silently ignored.
		}()

		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(errPipe)

			for scanner.Scan() {
				if !send(stderr, scanner.Text()) {
					return
				}
			}

			// TODO: Check scanner.Err() so stderr read errors are not silently ignored.
		}()

		err = session.Wait()
		wg.Wait() // wait for pipe readers to drain all remaining data
		close(finished)

		session.Close()
		close(stdout)
		close(stderr)

		// A cancelled session reports its own teardown as a failure. Report why
		// it really ended so the caller does not log a cancelled stream as a bug.
		if ctxErr := ctx.Err(); ctxErr != nil {
			done <- ctxErr
		} else {
			done <- err
		}

		// TODO: Consider wrapping errors so the caller knows whether the failure came from start/wait/stdout/stderr.
	}()

	return stdout, stderr, done
}

// Ensure that ServerConfig implements the net.Addr (optional, for testing)
var _ net.Addr = (*net.TCPAddr)(nil)

// TODO: This line only ensures that *net.TCPAddr implements net.Addr,
// not ServerConfig. If it serves no specific purpose, consider removing it.
