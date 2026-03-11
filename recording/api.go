package recording

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"encore.dev/beta/errs"
	"encore.dev/cron"
	"encore.dev/rlog"
	"encore.dev/storage/objects"
	"encore.dev/storage/sqldb"

	authpkg "encore.app/auth"
)

// GetRecordingURLParams contains parameters for getting a recording URL.
type GetRecordingURLParams struct {
	CallID string `query:"call_id"`
	Type   string `query:"type"` // "merged", "a", "b"
}

// GetRecordingURLResponse contains a presigned download URL.
type GetRecordingURLResponse struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// GetRecordingURL returns a presigned download URL for a recording.
//
//encore:api auth method=GET path=/recordings/url
func (s *Service) GetRecordingURL(ctx context.Context, p *GetRecordingURLParams) (*GetRecordingURLResponse, error) {
	if p.Type != "merged" && p.Type != "a" && p.Type != "b" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "type must be 'merged', 'a', or 'b'"}
	}
	if p.CallID == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "call_id is required"}
	}

	ad := authpkg.Data()
	if ad == nil {
		return nil, &errs.Error{Code: errs.Unauthenticated, Message: "not authenticated"}
	}

	// Look up call and recording keys.
	var userID int64
	var recordingKey, recordingAKey, recordingBKey *string
	var recordingStatus *string
	err := callbackDB.QueryRow(ctx,
		`SELECT user_id, recording_key, recording_a_key, recording_b_key, recording_status
		FROM callback_calls WHERE call_id = $1`, p.CallID,
	).Scan(&userID, &recordingKey, &recordingAKey, &recordingBKey, &recordingStatus)
	if err != nil {
		if err == sqldb.ErrNoRows {
			return nil, &errs.Error{Code: errs.NotFound, Message: "call not found"}
		}
		return nil, err
	}

	// Auth check: clients can only access their own recordings.
	if ad.Role != "admin" && userID != ad.UserID {
		return nil, &errs.Error{Code: errs.NotFound, Message: "call not found"}
	}

	if recordingStatus == nil || *recordingStatus != StatusReady {
		return nil, &errs.Error{Code: errs.NotFound, Message: "recording not available"}
	}

	// Select the appropriate key.
	var key *string
	switch p.Type {
	case "merged":
		key = recordingKey
	case "a":
		key = recordingAKey
	case "b":
		key = recordingBKey
	}
	if key == nil || *key == "" {
		return nil, &errs.Error{Code: errs.NotFound, Message: "recording not available"}
	}

	signed, err := RecordingsBucket.SignedDownloadURL(ctx, *key, objects.WithTTL(15*time.Minute))
	if err != nil {
		return nil, err
	}

	return &GetRecordingURLResponse{
		URL:       signed.URL,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}, nil
}

var _ = cron.NewJob("recording-cleanup", cron.JobConfig{
	Title:    "Clean up local recording files older than 24h",
	Every:    1 * cron.Hour,
	Endpoint: CleanupLocalRecordings,
})

// CleanupLocalRecordings removes local recording files older than 24 hours.
//
//encore:api private
func CleanupLocalRecordings(ctx context.Context) error {
	dir := "/var/lib/freeswitch/recordings"
	cutoff := time.Now().Add(-24 * time.Hour)
	deleted := 0

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible paths.
		}
		if info.IsDir() {
			return nil
		}
		if info.ModTime().Before(cutoff) {
			if removeErr := os.Remove(path); removeErr != nil {
				rlog.Warn("failed to remove old recording", "path", path, "error", removeErr)
			} else {
				deleted++
			}
		}
		return nil
	})
	if err != nil {
		rlog.Warn("error walking recordings directory", "error", err)
	}

	if deleted > 0 {
		rlog.Info("cleaned up old recording files", "count", deleted)
	}
	return nil
}
