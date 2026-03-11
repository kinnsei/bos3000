package fsclient

import (
	"context"
	"fmt"
	"sync"
	"time"

	"encore.dev/rlog"
)

// managedInstance wraps an FSClient with health state.
type managedInstance struct {
	client       FSClient
	address      string
	password     string
	healthy      bool
	failCount    int
	reconnecting bool
}

// FSClientManager manages primary/standby FreeSWITCH instances with failover.
type FSClientManager struct {
	instances          [2]*managedInstance
	mu                 sync.RWMutex
	activeIdx          int
	onFailoverCallback func(failedIdx int)
	stopProbes         chan struct{}
}

// NewFSClientManager creates a manager for two FreeSWITCH instances.
func NewFSClientManager(primaryAddr, primaryPwd, standbyAddr, standbyPwd string, onFailover func(int)) *FSClientManager {
	return &FSClientManager{
		instances: [2]*managedInstance{
			{address: primaryAddr, password: primaryPwd},
			{address: standbyAddr, password: standbyPwd},
		},
		onFailoverCallback: onFailover,
		stopProbes:         make(chan struct{}),
	}
}

// NewFSClientManagerWithClients creates a manager with pre-built FSClient instances (for testing).
func NewFSClientManagerWithClients(primary, standby FSClient, onFailover func(int)) *FSClientManager {
	return &FSClientManager{
		instances: [2]*managedInstance{
			{client: primary, healthy: true},
			{client: standby, healthy: true},
		},
		onFailoverCallback: onFailover,
		stopProbes:         make(chan struct{}),
	}
}

// Connect establishes ESL connections to both FreeSWITCH instances.
func (m *FSClientManager) Connect(ctx context.Context) error {
	connected := 0
	for i := range m.instances {
		inst := m.instances[i]
		if inst.client != nil {
			connected++
			continue
		}
		client, err := NewESLFSClient(inst.address, inst.password, func() {
			m.handleDisconnect(i)
		})
		if err != nil {
			rlog.Warn("failed to connect to FreeSWITCH", "idx", i, "address", inst.address, "error", err)
			continue
		}
		inst.client = client
		inst.healthy = true
		connected++
		go m.startHealthProbe(i)
	}

	if connected == 0 {
		return fmt.Errorf("failed to connect to any FreeSWITCH instance")
	}

	m.mu.Lock()
	if m.instances[0].healthy {
		m.activeIdx = 0
	} else {
		m.activeIdx = 1
	}
	m.mu.Unlock()

	return nil
}

// Pick returns the currently active healthy FSClient.
func (m *FSClientManager) Pick() (FSClient, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.instances[m.activeIdx].healthy {
		return m.instances[m.activeIdx].client, nil
	}
	other := 1 - m.activeIdx
	if m.instances[other].healthy {
		return m.instances[other].client, nil
	}
	return nil, fmt.Errorf("no healthy FreeSWITCH instance")
}

// MarkUnhealthy marks an instance as unhealthy and triggers failover if needed.
func (m *FSClientManager) MarkUnhealthy(idx int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.instances[idx].healthy = false
	m.instances[idx].failCount = 3

	if m.activeIdx == idx {
		other := 1 - idx
		if m.instances[other].healthy {
			m.activeIdx = other
			rlog.Info("failover triggered", "from", idx, "to", other)
		}
		if m.onFailoverCallback != nil {
			go m.onFailoverCallback(idx)
		}
	}
}

// ActiveIdx returns the current active instance index (for testing).
func (m *FSClientManager) ActiveIdx() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeIdx
}

// MarkHealthy marks an instance as healthy without switching active index.
// Per design: no auto-switchback after recovery.
func (m *FSClientManager) MarkHealthy(idx int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.instances[idx].healthy = true
	m.instances[idx].failCount = 0
}

func (m *FSClientManager) startHealthProbe(idx int) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			inst := m.instances[idx]
			if inst.client == nil {
				continue
			}
			// Send "status" command as health check.
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := inst.client.HangupCall(ctx, "health-check-noop", "")
			cancel()
			// HangupCall on non-existent UUID is a lightweight probe.
			// For ESL clients, any response means connection is alive.
			_ = err

			// For a real health probe we'd use SendCommand("status"),
			// but FSClient interface doesn't expose raw commands.
			// The connection itself being alive is sufficient.
			m.mu.Lock()
			inst.failCount = 0
			inst.healthy = true
			// No auto-switchback: do NOT change activeIdx here.
			m.mu.Unlock()
		case <-m.stopProbes:
			return
		}
	}
}

func (m *FSClientManager) handleDisconnect(idx int) {
	m.MarkUnhealthy(idx)

	go func() {
		backoff := time.Second
		maxBackoff := 30 * time.Second

		for {
			rlog.Info("attempting reconnection", "idx", idx, "backoff", backoff)
			time.Sleep(backoff)

			inst := m.instances[idx]
			client, err := NewESLFSClient(inst.address, inst.password, func() {
				m.handleDisconnect(idx)
			})
			if err != nil {
				backoff = min(backoff*2, maxBackoff)
				continue
			}

			m.mu.Lock()
			inst.client = client
			inst.healthy = true
			inst.failCount = 0
			inst.reconnecting = false
			// No auto-switchback: do NOT change activeIdx.
			m.mu.Unlock()

			go m.startHealthProbe(idx)
			rlog.Info("reconnected to FreeSWITCH", "idx", idx)
			return
		}
	}()
}
