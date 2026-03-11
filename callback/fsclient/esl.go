package fsclient

import (
	"context"
	"fmt"
	"strings"
	"sync"

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

// NewESLFSClient creates an ESL client connected to FreeSWITCH.
func NewESLFSClient(address, password string, onDisconnect func()) (*ESLFSClient, error) {
	conn, err := eslgo.Dial(address, password, onDisconnect)
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
	_, err = conn.SendCommand(ctx, rawCommand("event plain CHANNEL_ANSWER CHANNEL_BRIDGE CHANNEL_HANGUP"))
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
	aLeg := eslgo.Leg{
		CallURL: fmt.Sprintf("sofia/gateway/%s/%s", params.GatewayIP, params.Number),
	}
	parkApp := eslgo.Leg{CallURL: "&park()"}
	vars := map[string]string{
		"origination_uuid":              uuid,
		"origination_caller_id_number":  params.CallerID,
	}

	resp, err := c.conn.OriginateCall(ctx, true, aLeg, parkApp, vars)
	if err != nil {
		return "", fmt.Errorf("originate a-leg: %w", err)
	}
	if !resp.IsOk() {
		return "", fmt.Errorf("originate a-leg: %s", resp.GetReply())
	}
	return uuid, nil
}

func (c *ESLFSClient) OriginateBLegAndBridge(ctx context.Context, aUUID string, params OriginateParams) (string, error) {
	bUUID := params.CallID + "-b"
	bLeg := eslgo.Leg{
		CallURL: fmt.Sprintf("sofia/gateway/%s/%s", params.GatewayIP, params.Number),
	}
	bridgeApp := eslgo.Leg{CallURL: fmt.Sprintf("&bridge(%s)", aUUID)}
	vars := map[string]string{
		"origination_uuid": bUUID,
	}

	resp, err := c.conn.OriginateCall(ctx, true, bLeg, bridgeApp, vars)
	if err != nil {
		return "", fmt.Errorf("originate b-leg: %w", err)
	}
	if !resp.IsOk() {
		return "", fmt.Errorf("originate b-leg: %s", resp.GetReply())
	}
	return bUUID, nil
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
	if name != "CHANNEL_ANSWER" && name != "CHANNEL_BRIDGE" && name != "CHANNEL_HANGUP" {
		return
	}

	uuid := event.GetHeader("Unique-ID")
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
