package fsclient

import "context"

// FSClient defines the interface for interacting with FreeSWITCH.
// In Phase 2 (Mock), this is implemented by MockFSClient.
// In Phase 3, this will be implemented by ESLFSClient.
type FSClient interface {
	OriginateALeg(ctx context.Context, params OriginateParams) (uuid string, err error)
	OriginateBLegAndBridge(ctx context.Context, aUUID string, params OriginateParams) (bUUID string, err error)
	BridgeCall(ctx context.Context, aUUID, bUUID string) error
	HangupCall(ctx context.Context, uuid string, cause string) error
	StartRecording(ctx context.Context, uuid string, callID string, leg string) error
	StopRecording(ctx context.Context, uuid string, callID string, leg string) error
	RegisterEventHandler(eventName string, handler func(CallEvent))
}
