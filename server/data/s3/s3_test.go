package s3

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewConfigDefaults(t *testing.T) {
	config := NewConfig(make(map[string]any))
	require.NotNil(t, config, "invalid nil config")
	require.Equal(t, int64(16*1024*1024), config.PartSize, "invalid default part size")
	require.Equal(t, 4, config.PartUploadConcurrency, "invalid default part upload concurrency")
	require.False(t, config.SendContentMd5, "SendContentMd5 should be disabled by default")
	require.Equal(t, "auto", config.BucketLookup, "BucketLookup should default to auto")
}

func TestNewConfigWithSendContentMd5(t *testing.T) {
	config := NewConfig(map[string]any{
		"SendContentMd5": true,
	})
	require.True(t, config.SendContentMd5, "invalid SendContentMd5 override")
}

func TestNewConfigWithPathStyle(t *testing.T) {
	config := NewConfig(map[string]any{
		"BucketLookup": "path",
	})
	require.Equal(t, "path", config.BucketLookup, "invalid BucketLookup override")
}

func TestNewConfigWithPartUploadConcurrency(t *testing.T) {
	config := NewConfig(map[string]any{
		"PartUploadConcurrency": 4,
	})
	require.Equal(t, 4, config.PartUploadConcurrency, "invalid PartUploadConcurrency override")
}

func TestNewPutObjectOptions(t *testing.T) {
	backend := &Backend{
		config: &Config{
			SendContentMd5: true,
		},
	}

	opts := backend.newPutObjectOptions("application/octet-stream")
	require.Equal(t, "application/octet-stream", opts.ContentType, "invalid content type")
	require.True(t, opts.SendContentMd5, "invalid send content md5 option")
}

func validConfig() *Config {
	return &Config{
		Endpoint:              "s3.example.com",
		AccessKeyID:           "test",
		SecretAccessKey:       "test",
		Bucket:                "test",
		Location:              "us-east-1",
		PartSize:              16 * 1024 * 1024,
		PartUploadConcurrency: 4,
	}
}

func TestValidateRejectsNegativePartSize(t *testing.T) {
	config := validConfig()
	config.PartSize = -1
	err := config.Validate()
	require.Error(t, err, "negative PartSize should be rejected")
	require.Contains(t, err.Error(), "part size")
}

func TestValidateRejectsZeroPartUploadConcurrency(t *testing.T) {
	config := validConfig()
	config.PartUploadConcurrency = 0
	err := config.Validate()
	require.Error(t, err, "zero PartUploadConcurrency should be rejected")
	require.Contains(t, err.Error(), "concurrency")
}

func TestValidateRejectsNegativePartUploadConcurrency(t *testing.T) {
	config := validConfig()
	config.PartUploadConcurrency = -1
	err := config.Validate()
	require.Error(t, err, "negative PartUploadConcurrency should be rejected")
	require.Contains(t, err.Error(), "concurrency")
}
