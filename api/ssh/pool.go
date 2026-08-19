package ssh

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	// poolIdleTTL bounds how long an unused connection is kept alive. A server
	// somebody is watching keeps its connection warm between dashboard polls; a
	// server nobody is watching gives it back instead of holding it forever.
	poolIdleTTL = 5 * time.Minute
	// poolSweepPeriod is how often idle connections are looked for. Sweeping
	// well under the TTL keeps the actual idle time close to the configured one.
	poolSweepPeriod = time.Minute
)

// ErrPoolClosed is returned once the pool has been shut down.
var ErrPoolClosed = errors.New("ssh connection pool is closed")

// DefaultPool backs the pooled connections used across the process.
var DefaultPool = NewPool()

// Pool reuses authenticated SSH connections across requests.
//
// Monitoring reads tunnel their HTTP traffic through SSH, so without reuse every
// dashboard poll would pay a fresh TCP connect and key exchange. One gossh
// client multiplexes independent channels, so a single pooled connection serves
// concurrent callers safely.
//
// Connections handed out by the pool belong to the pool. Callers reach them
// through Use and must never Close them. The pool is meant for tunneling, not
// for command execution: DetectDocker mutates the client it is called on, which
// is not safe on a connection shared between goroutines.
type Pool struct {
	mu      sync.Mutex
	entries map[string]*pooledConn
	closed  bool
	done    chan struct{}
}

type pooledConn struct {
	client   *Client
	lastUsed time.Time
	// inUse keeps a connection from being closed underneath an in-flight
	// request.
	inUse int
	// detached marks a connection already dropped from the map. Whoever
	// releases it last closes it.
	detached bool
}

// lease is one caller's borrowed hold on a pooled connection.
type lease struct {
	pool   *Pool
	key    string
	entry  *pooledConn
	client *Client
	// fresh marks a connection dialed for this lease, which is worth knowing on
	// failure: a brand new connection is not worth retrying.
	fresh bool
}

func NewPool() *Pool {
	pool := &Pool{entries: map[string]*pooledConn{}, done: make(chan struct{})}
	go pool.sweep()
	return pool
}

// Use runs fn against a pooled connection to cfg, dialing one if the pool has
// none.
//
// A pooled connection can be severed between requests by anything from a reboot
// to an idle timeout on the far end, and the break only surfaces on the next
// use. So when fn fails on a reused connection that turns out to be dead, the
// connection is dropped and the call retried once on a fresh one.
func Use[T any](pool *Pool, cfg ServerConfig, fn func(*Client) (T, error)) (T, error) {
	var zero T
	borrowed, err := pool.acquire(cfg)
	if err != nil {
		return zero, err
	}
	result, err := fn(borrowed.client)
	dead := err != nil && !borrowed.client.alive()
	retryable := dead && !borrowed.fresh
	borrowed.release(dead)
	if !retryable {
		return result, err
	}

	retried, dialErr := pool.acquire(cfg)
	if dialErr != nil {
		// The reconnect failing says less about the request than the original
		// error does, so report that one.
		return zero, err
	}
	result, err = fn(retried.client)
	retried.release(err != nil && !retried.client.alive())
	return result, err
}

// Close drops every pooled connection and stops the sweeper. In-flight requests
// are cut short, so this belongs in shutdown and nowhere else.
func (p *Pool) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	close(p.done)
	stale := p.detachAll()
	p.mu.Unlock()
	closeAll(stale)
}

func (p *Pool) acquire(cfg ServerConfig) (*lease, error) {
	key := poolKey(cfg)

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, ErrPoolClosed
	}
	if entry, ok := p.entries[key]; ok {
		p.borrow(entry)
		p.mu.Unlock()
		return &lease{pool: p, key: key, entry: entry, client: entry.client}, nil
	}
	p.mu.Unlock()

	// Dialing outside the lock: a handshake against an unreachable server can
	// take the full timeout, and holding the lock for it would stall reads to
	// every other server.
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		client.Close()
		return nil, ErrPoolClosed
	}
	// Another caller may have dialed the same server while this one was
	// handshaking. Keep the connection already in the pool and let this one
	// stand alone, closing as soon as its lease ends.
	if entry, ok := p.entries[key]; ok {
		p.borrow(entry)
		p.mu.Unlock()
		client.Close()
		return &lease{pool: p, key: key, entry: entry, client: entry.client}, nil
	}
	entry := &pooledConn{client: client, lastUsed: time.Now(), inUse: 1}
	p.entries[key] = entry
	p.mu.Unlock()
	return &lease{pool: p, key: key, entry: entry, client: client, fresh: true}, nil
}

// borrow records a lease on entry. Callers must hold p.mu.
func (p *Pool) borrow(entry *pooledConn) {
	entry.inUse++
	entry.lastUsed = time.Now()
}

// detachAll empties the map and marks every entry detached. Callers must hold
// p.mu, and must close the returned entries once it is released.
func (p *Pool) detachAll() []*pooledConn {
	stale := make([]*pooledConn, 0, len(p.entries))
	for key, entry := range p.entries {
		delete(p.entries, key)
		entry.detached = true
		stale = append(stale, entry)
	}
	return stale
}

func (l *lease) release(broken bool) {
	pool := l.pool
	pool.mu.Lock()
	l.entry.inUse--
	l.entry.lastUsed = time.Now()
	if broken && !l.entry.detached {
		if pool.entries[l.key] == l.entry {
			delete(pool.entries, l.key)
		}
		l.entry.detached = true
	}
	orphaned := l.entry.detached && l.entry.inUse == 0
	pool.mu.Unlock()
	if orphaned {
		l.entry.client.Close()
	}
}

func (p *Pool) sweep() {
	ticker := time.NewTicker(poolSweepPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-p.done:
			return
		case <-ticker.C:
			p.evictIdle(time.Now())
		}
	}
}

func (p *Pool) evictIdle(now time.Time) {
	var stale []*pooledConn
	p.mu.Lock()
	for key, entry := range p.entries {
		if entry.inUse > 0 || now.Sub(entry.lastUsed) <= poolIdleTTL {
			continue
		}
		delete(p.entries, key)
		entry.detached = true
		stale = append(stale, entry)
	}
	p.mu.Unlock()
	closeAll(stale)
}

func closeAll(entries []*pooledConn) {
	for _, entry := range entries {
		entry.client.Close()
	}
}

// poolKey identifies one set of credentials against one address. The key
// material is hashed in so a rotated key never reuses a connection that was
// authenticated with the old one.
func poolKey(cfg ServerConfig) string {
	sum := sha256.Sum256([]byte(cfg.PrivateKey))
	return fmt.Sprintf("%s@%s:%d/%s", cfg.User, cfg.Host, cfg.Port, hex.EncodeToString(sum[:8]))
}
