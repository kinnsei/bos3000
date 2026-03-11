package recording

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestMergeRecordings_MissingFile(t *testing.T) {
	err := MergeRecordings(context.Background(), "/nonexistent/a.wav", "/nonexistent/b.wav", "/tmp/out.mp3")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestMergeRecordings_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	aPath := filepath.Join(dir, "a.wav")
	bPath := filepath.Join(dir, "b.wav")

	// Create empty files.
	os.WriteFile(aPath, []byte{}, 0644)
	os.WriteFile(bPath, []byte("data"), 0644)

	err := MergeRecordings(context.Background(), aPath, bPath, filepath.Join(dir, "out.mp3"))
	if err == nil {
		t.Fatal("expected error for empty file")
	}
}

func TestMergeRecordings_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffmpeg test in short mode")
	}

	dir := t.TempDir()
	aPath := filepath.Join(dir, "a.wav")
	bPath := filepath.Join(dir, "b.wav")

	// Create minimal non-empty files (not valid WAV, but triggers ffmpeg which then gets cancelled).
	os.WriteFile(aPath, []byte("RIFF----WAVEfmt "), 0644)
	os.WriteFile(bPath, []byte("RIFF----WAVEfmt "), 0644)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	err := MergeRecordings(ctx, aPath, bPath, filepath.Join(dir, "out.mp3"))
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
}
