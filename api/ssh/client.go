package ssh

import (
	"bufio"
	"fmt"
	"net"

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
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	client, err := gossh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("Cannot connect to %s: %w", addr, err)
	}

	return &Client{client: client}, err
}

func (c *Client) Close() {
	c.client.Close()
}

// StreamCommand - run command and return the output line by line via channel
func (c *Client) StreamCommand(command string) (<-chan string, <-chan string, <-chan error) {
	stdout := make(chan string)
	stderr := make(chan string)
	done := make(chan error, 1)

	go func() {
		defer close(stdout)
		defer close(stderr)

		session, err := c.client.NewSession()
		if err != nil {
			done <- err
			return
		}
		defer session.Close()

		outPipe, _ := session.StdoutPipe()
		errPipe, _ := session.StderrPipe()

		session.Start(command)

		// read stdout
		go func() {
			scanner := bufio.NewScanner(outPipe)
			for scanner.Scan() {
				stdout <- scanner.Text()
			}
		}()

		// read stderr
		go func() {
			scanner := bufio.NewScanner(errPipe)
			for scanner.Scan() {
				stderr <- scanner.Text()
			}
		}()

		done <- session.Wait()
	}()

	return stdout, stderr, done
}

// Ensure that ServerConfig implements the net.Addr (optional, for testing)
var _ net.Addr = (*net.TCPAddr)(nil)
