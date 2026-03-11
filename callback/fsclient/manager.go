package fsclient

import (
	"context"
	"fmt"
	"sync"
	"time"

	"encore.dev/rlog"
)

// FSClientManager manages a single FreeSWITCH instance with reconnection.
type FSClientManager struct {
	client     FSClient
	address    string
	password   string
	healthy    bool
	mu         sync.RWMutex
	onDisconnect func()
	stopProbes chan struct{}
}

// NewFSClientManager creates a manager for a single FreeSWITCH instance.
func NewFSClientManager(addr, pwd string, onDisconnect func()) *FSClientManager {
	return &FSClientManager{
		address:      addr,
		password:     pwd,
		onDisconnect: onDisconnect,
		stopProbes:   make(chan struct{}),
	}
}

// NewFSClientManagerWithClient creates a manager with a pre-built FSClient (for testing).
func NewFSClientManagerWithClient(client FSClient, onDisconnect func()) *FSClientManager {
	return &FSClientManager{
		client:       client,
		healthy:      true,
		onDisconnect: onDisconnect,
		stopProbes:   make(chan struct{}),
	}
}

// Connect establishes ESL connection to FreeSWITCH.
func (m *FSClientManager) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client != nil {
		return nil
	}

	client, err := NewESLFSClient(m.address, m.password, func() {
		m.handleDisconnect()
	})
	if err != nil {
		return fmt.Errorf("connect to FreeSWITCH %s: %w", m.address, err)
	}

	m.client = client
	m.healthy = true
	go m.startHealthProbe()
	return nil
}

// Pick returns the FSClient if healthy.
func (m *FSClientManager) Pick() (FSClient, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.healthy && m.client != nil {
		return m.client, nil
	}
	return nil, fmt.Errorf("no healthy FreeSWITCH instance")
}

func (m *FSClientManager) startHealthProbe() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.mu.RLock()
			c := m.client
			m.mu.RUnlock()
			if c == nil {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = c.HangupCall(ctx, "health-check-noop", "")
			cancel()
		case <-m.stopProbes:
			return
		}
	}
}

func (m *FSClientManager) handleDisconnect() {
	m.mu.Lock()
	m.healthy = false
	m.mu.Unlock()

	if m.onDisconnect != nil {
		go m.onDisconnect()
	}

	go func() {
		backoff := time.Second
		maxBackoff := 30 * time.Second

		for {
			rlog.Info("attempting reconnection", "address", m.address, "backoff", backoff)
			time.Sleep(backoff)

			client, err := NewESLFSClient(m.address, m.password, func() {
				m.handleDisconnect()
			})
			if err != nil {
				backoff = min(backoff*2, maxBackoff)
				continue
			}

			m.mu.Lock()
			m.client = client
			m.healthy = true
			m.mu.Unlock()

			go m.startHealthProbe()
			rlog.Info("reconnected to FreeSWITCH", "address", m.address)
			return
		}
	}()
}
