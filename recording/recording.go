package recording

import (
	"encore.dev/pubsub"
	"encore.dev/storage/objects"
)

// RecordingsBucket stores merged and individual leg recording files.
// INFRASTRUCTURE NOTE: Per CONTEXT.md, recordings must be retained for 90 days
// then auto-deleted. Encore Object Storage does not expose lifecycle policy
// configuration in code. The 90-day lifecycle rule must be configured at the
// infrastructure level:
//   - Encore Cloud: Set via Encore Cloud dashboard bucket settings
//   - AWS S3: Configure lifecycle rule on the provisioned bucket
//   - GCP GCS: Configure lifecycle rule on the provisioned bucket
var RecordingsBucket = objects.NewBucket("recordings", objects.BucketConfig{
	Versioned: false,
})

// RecordingMergeTopic triggers the merge worker when both legs finish recording.
var RecordingMergeTopic = pubsub.NewTopic[*RecordingMergeEvent]("recording-merge", pubsub.TopicConfig{
	DeliveryGuarantee: pubsub.AtLeastOnce,
})

//encore:service
type Service struct{}

func initService() (*Service, error) {
	return &Service{}, nil
}
