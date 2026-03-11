package recording

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"encore.dev/pubsub"
	"encore.dev/rlog"
	"encore.dev/storage/sqldb"
)

var callbackDB = sqldb.Named("callback")

var _ = pubsub.NewSubscription(
	RecordingMergeTopic, "recording-merge-worker",
	pubsub.SubscriptionConfig[*RecordingMergeEvent]{
		Handler: pubsub.MethodHandler((*Service).HandleMerge),
		RetryPolicy: &pubsub.RetryPolicy{
			MinBackoff: 30 * time.Second,
			MaxBackoff: 5 * time.Minute,
			MaxRetries: 3,
		},
	},
)

// HandleMerge processes a recording merge event.
func (s *Service) HandleMerge(ctx context.Context, event *RecordingMergeEvent) error {
	outputPath := fmt.Sprintf("/tmp/%s_merged.mp3", event.CallID)

	// Merge recordings.
	if err := MergeRecordings(ctx, event.AFilePath, event.BFilePath, outputPath); err != nil {
		_, _ = callbackDB.Exec(ctx,
			"UPDATE callback_calls SET recording_status = $1 WHERE call_id = $2",
			StatusFailed, event.CallID)
		return fmt.Errorf("merge failed for %s: %w", event.CallID, err)
	}

	// Upload merged file.
	mergedKey := fmt.Sprintf("recordings/%d/%s/%s_merged.mp3", event.CustomerID, event.Date, event.CallID)
	if err := uploadFile(ctx, mergedKey, outputPath); err != nil {
		return fmt.Errorf("upload merged: %w", err)
	}

	// Upload A-leg WAV.
	aKey := fmt.Sprintf("recordings/%d/%s/%s_a.wav", event.CustomerID, event.Date, event.CallID)
	if err := uploadFile(ctx, aKey, event.AFilePath); err != nil {
		rlog.Warn("failed to upload a-leg", "error", err)
	}

	// Upload B-leg WAV.
	bKey := fmt.Sprintf("recordings/%d/%s/%s_b.wav", event.CustomerID, event.Date, event.CallID)
	if err := uploadFile(ctx, bKey, event.BFilePath); err != nil {
		rlog.Warn("failed to upload b-leg", "error", err)
	}

	// Update callback record.
	_, err := callbackDB.Exec(ctx,
		`UPDATE callback_calls SET
			recording_key = $1, recording_a_key = $2, recording_b_key = $3,
			recording_status = $4
		WHERE call_id = $5`,
		mergedKey, aKey, bKey, StatusReady, event.CallID)
	if err != nil {
		return fmt.Errorf("update recording status: %w", err)
	}

	// Clean up local merged file. Original WAVs cleaned by 24h cron.
	os.Remove(outputPath)

	return nil
}

// uploadFile reads a local file and uploads it to the recordings bucket.
func uploadFile(ctx context.Context, key, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read %s: %w", filePath, err)
	}

	w := RecordingsBucket.Upload(ctx, key)
	if _, err := io.Copy(w, bytes.NewReader(data)); err != nil {
		w.Abort(err)
		return fmt.Errorf("write %s: %w", key, err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close %s: %w", key, err)
	}
	return nil
}
