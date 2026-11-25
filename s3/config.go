package s3

import "time"

type Config struct {
	Enabled         bool
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	ForcePathStyle  bool
	Timeout         time.Duration
}
