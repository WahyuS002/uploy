package ssh

import (
	"bufio"
	"fmt"
	"net"
	"sync"

	gossh "golang.org/x/crypto/ssh"
)

type Client struct {
	client *gossh.Client
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
		return nil, fmt.Errorf("Private key is not valid: %w", err)
	}

	config := &gossh.ClientConfig{
		User:            cfg.User,
		Auth:            []gossh.AuthMethod{gossh.PublicKeys(signer)},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),

		// TODO: Replace InsecureIgnoreHostKey with known_hosts or pinned host key verification.
		// TODO: Add Timeout to ClientConfig to prevent dial from hanging forever.
		// TODO: Consider restricting SSH algorithms to secure SupportedAlgorithms().
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	client, err := gossh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("Cannot connect to %s: %w", addr, err)
	}

	// TODO: Return nil explicitly for clarity.
	return &Client{client: client}, err
}

func (c *Client) Close() {
	// TODO: Consider returning an error from Close() so the caller can handle failures.
	c.client.Close()
}

// StreamCommand - run command and return the output line by line via channel
func (c *Client) StreamCommand(command string) (<-chan string, <-chan string, <-chan error) {
	stdout := make(chan string)
	stderr := make(chan string)
	done := make(chan error, 1)

	// TODO: Add context.Context so the caller can cancel the command.
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
		// TODO: Consider using defer session.Close() immediately after session creation succeeds.

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

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(outPipe)

			// TODO: Increase the scanner buffer if log output may contain long lines (> 64KB).
			// TODO: Consider using bufio.Reader instead if long-line handling needs to be safer later.

			for scanner.Scan() {
				stdout <- scanner.Text()

				// TODO: Be careful: sending to an unbuffered channel can block
				// if the consumer is slow or stops reading.
			}

			// TODO: Check scanner.Err() so stdout read errors are not silently ignored.
		}()

		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(errPipe)

			// TODO: Increase the scanner buffer if log output may contain long lines (> 64KB).
			// TODO: Consider using bufio.Reader instead if long-line handling needs to be safer later.

			for scanner.Scan() {
				stderr <- scanner.Text()

				// TODO: Be careful: sending to an unbuffered channel can block
				// if the consumer is slow or stops reading.
			}

			// TODO: Check scanner.Err() so stderr read errors are not silently ignored.
		}()

		err = session.Wait()
		wg.Wait() // wait for pipe readers to drain all remaining data

		// TODO: Ensure there is no goroutine leak if the caller stops consuming channels before the command finishes.
		// TODO: Consider adding a cancellation path to stop the session when the client disconnects, times out, or the deploy is canceled.

		session.Close()
		close(stdout)
		close(stderr)
		done <- err

		// TODO: Consider closing done as well if you want the API pattern to be more consistent.
		// TODO: Consider wrapping errors so the caller knows whether the failure came from start/wait/stdout/stderr.
	}()

	return stdout, stderr, done
}

// Ensure that ServerConfig implements the net.Addr (optional, for testing)
var _ net.Addr = (*net.TCPAddr)(nil)

// TODO: This line only ensures that *net.TCPAddr implements net.Addr,
// not ServerConfig. If it serves no specific purpose, consider removing it.
