package recording

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// MergeRecordings combines A-leg and B-leg WAV files into a stereo MP3.
func MergeRecordings(ctx context.Context, aPath, bPath, outputPath string) error {
	// Validate input files exist and are non-empty.
	for _, path := range []string{aPath, bPath} {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("recording file not found: %s: %w", path, err)
		}
		if info.Size() == 0 {
			return fmt.Errorf("recording file is empty: %s", path)
		}
	}

	mergeCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(mergeCtx, "ffmpeg",
		"-y",
		"-i", aPath,
		"-i", bPath,
		"-filter_complex", "[0:a][1:a]amerge=inputs=2[a]",
		"-map", "[a]",
		"-ac", "2",
		"-codec:a", "libmp3lame",
		"-q:a", "2",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg merge failed: %w: %s", err, string(output))
	}
	return nil
}
