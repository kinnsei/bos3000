package recording

const (
	StatusRecording = "recording"
	StatusMerging   = "merging"
	StatusReady     = "ready"
	StatusFailed    = "failed"
)

// RecordingMergeEvent is published when both call legs finish recording.
type RecordingMergeEvent struct {
	CallID     string `json:"call_id"`
	CustomerID int64  `json:"customer_id"`
	AFilePath  string `json:"a_file_path"`
	BFilePath  string `json:"b_file_path"`
	Date       string `json:"date"` // YYYY-MM-DD
}
