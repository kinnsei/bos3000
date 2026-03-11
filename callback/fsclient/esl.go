package fsclient

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"encore.dev/rlog"
	"github.com/percipia/eslgo"
	"github.com/percipia/eslgo/command"
)

// Compile-time interface check.
var _ FSClient = (*ESLFSClient)(nil)

// ESLFSClient implements FSClient using eslgo for real FreeSWITCH connections.
type ESLFSClient struct {
	conn         *eslgo.Conn
	handlers     map[string][]func(CallEvent)
	mu           sync.RWMutex
	address      string
	password     string
	onDisconnect func()
}

// dialWithTimeout wraps eslgo.Dial with a timeout to prevent blocking on
// unreachable instances (eslgo.Dial has no built-in timeout).
func dialWithTimeout(address, password string, onDisconnect func(), timeout time.Duration) (*eslgo.Conn, error) {
	// Pre-check TCP reachability to fail fast.
	tcpConn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return nil, fmt.Errorf("tcp connect %s: %w", address, err)
	}
	tcpConn.Close()

	type result struct {
		conn *eslgo.Conn
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		c, e := eslgo.Dial(address, password, onDisconnect)
		ch <- result{c, e}
	}()

	select {
	case r := <-ch:
		return r.conn, r.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("esl dial %s: timeout after %s", address, timeout)
	}
}

// NewESLFSClient creates an ESL client connected to FreeSWITCH.
func NewESLFSClient(address, password string, onDisconnect func()) (*ESLFSClient, error) {
	conn, err := dialWithTimeout(address, password, onDisconnect, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("esl dial %s: %w", address, err)
	}

	client := &ESLFSClient{
		conn:         conn,
		handlers:     make(map[string][]func(CallEvent)),
		address:      address,
		password:     password,
		onDisconnect: onDisconnect,
	}

	// Subscribe to relevant events.
	ctx := context.Background()
	_, err = conn.SendCommand(ctx, eslCommand("event plain CHANNEL_ANSWER CHANNEL_BRIDGE CHANNEL_HANGUP CHANNEL_HANGUP_COMPLETE"))
	if err != nil {
		conn.ExitAndClose()
		return nil, fmt.Errorf("esl subscribe events: %w", err)
	}

	// Register global event listener.
	conn.RegisterEventListener(eslgo.EventListenAll, client.dispatchEvent)

	return client, nil
}

func (c *ESLFSClient) OriginateALeg(ctx context.Context, params OriginateParams) (string, error) {
	uuid := params.CallID + "-a"
	callerID := params.CallerID
	if callerID == "" {
		callerID = params.Number
	}
	cmd := fmt.Sprintf("originate {origination_uuid=%s,origination_caller_id_number=%s}sofia/gateway/%s/%s &park()",
		uuid, callerID, params.GatewayIP, params.Number)

	rlog.Info("originate a-leg", "cmd", cmd)
	resp, err := c.conn.SendCommand(ctx, rawCommand(cmd))
	if err != nil {
		return "", fmt.Errorf("originate a-leg: %w", err)
	}
	reply := resp.GetReply()
	if !strings.HasPrefix(reply, "+OK") {
		return "", fmt.Errorf("originate a-leg: %s", reply)
	}
	return uuid, nil
}

func (c *ESLFSClient) OriginateBLegAndBridge(ctx context.Context, aUUID string, params OriginateParams) (string, error) {
	bUUID := params.CallID + "-b"
	callerID := params.CallerID
	if callerID == "" {
		callerID = params.Number
	}
	// Originate B-leg to park, then state machine will bridge after B answers.
	cmd := fmt.Sprintf("originate {origination_uuid=%s,origination_caller_id_number=%s}sofia/gateway/%s/%s &park()",
		bUUID, callerID, params.GatewayIP, params.Number)

	rlog.Info("originate b-leg", "cmd", cmd)
	resp, err := c.conn.SendCommand(ctx, rawCommand(cmd))
	if err != nil {
		return "", fmt.Errorf("originate b-leg: %w", err)
	}
	reply := resp.GetReply()
	if !strings.HasPrefix(reply, "+OK") {
		return "", fmt.Errorf("originate b-leg: %s", reply)
	}
	return bUUID, nil
}

func (c *ESLFSClient) BridgeCall(ctx context.Context, aUUID, bUUID string) error {
	cmd := fmt.Sprintf("uuid_bridge %s %s", aUUID, bUUID)
	rlog.Info("bridge legs", "cmd", cmd)
	resp, err := c.conn.SendCommand(ctx, rawCommand(cmd))
	if err != nil {
		return fmt.Errorf("uuid_bridge: %w", err)
	}
	reply := resp.GetReply()
	if !strings.HasPrefix(reply, "+OK") {
		return fmt.Errorf("uuid_bridge: %s", reply)
	}
	return nil
}

func (c *ESLFSClient) HangupCall(ctx context.Context, uuid string, cause string) error {
	return c.conn.HangupCall(ctx, uuid, cause)
}

func (c *ESLFSClient) StartRecording(ctx context.Context, uuid string, callID string, leg string) error {
	path := fmt.Sprintf("/var/lib/freeswitch/recordings/%s_%s.wav", callID, leg)
	cmd := fmt.Sprintf("uuid_record %s start %s", uuid, path)
	_, err := c.conn.SendCommand(ctx, rawCommand(cmd))
	return err
}

func (c *ESLFSClient) StopRecording(ctx context.Context, uuid string, callID string, leg string) error {
	path := fmt.Sprintf("/var/lib/freeswitch/recordings/%s_%s.wav", callID, leg)
	cmd := fmt.Sprintf("uuid_record %s stop %s", uuid, path)
	_, err := c.conn.SendCommand(ctx, rawCommand(cmd))
	return err
}

func (c *ESLFSClient) RegisterEventHandler(eventName string, handler func(CallEvent)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[eventName] = append(c.handlers[eventName], handler)
}

func (c *ESLFSClient) dispatchEvent(event *eslgo.Event) {
	name := event.GetName()
	uuid := event.GetHeader("Unique-ID")
	rlog.Debug("esl event received", "event", name, "uuid", uuid)

	// Treat CHANNEL_HANGUP_COMPLETE as CHANNEL_HANGUP for dispatch purposes.
	if name == "CHANNEL_HANGUP_COMPLETE" {
		name = "CHANNEL_HANGUP"
	}
	if name != "CHANNEL_ANSWER" && name != "CHANNEL_BRIDGE" && name != "CHANNEL_HANGUP" {
		return
	}
	if uuid == "" {
		return
	}

	// Determine leg from UUID suffix.
	leg := "A"
	if strings.HasSuffix(uuid, "-b") {
		leg = "B"
	}

	// Extract callID by removing leg suffix.
	callID := uuid
	if strings.HasSuffix(uuid, "-a") || strings.HasSuffix(uuid, "-b") {
		callID = uuid[:len(uuid)-2]
	}

	ce := CallEvent{
		CallID:    callID,
		UUID:      uuid,
		Leg:       leg,
		EventName: name,
		Timestamp: time.Now(),
	}

	if name == "CHANNEL_HANGUP" {
		ce.HangupCause = event.GetHeader("Hangup-Cause")
	}

	c.mu.RLock()
	handlers := c.handlers[name]
	c.mu.RUnlock()

	if len(handlers) == 0 {
		rlog.Warn("orphan ESL event, no handlers registered",
			"uuid", uuid, "event", name, "call_id", callID)
		return
	}

	for _, h := range handlers {
		h(ce)
	}
}

// rawCommand implements command.Command for sending raw ESL API commands.
type rawCommand string

var _ command.Command = rawCommand("")

func (r rawCommand) BuildMessage() string {
	return "api " + string(r)
}

// eslCommand implements command.Command for native ESL protocol commands
// (like "event", "filter") that must NOT be prefixed with "api".
type eslCommand string

var _ command.Command = eslCommand("")

func (e eslCommand) BuildMessage() string {
	return string(e)
}
