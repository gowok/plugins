package s3

import "time"

type Config struct {
	Enabled         bool
	Region          string
	Endpoint        string
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Bucket          string
	ForcePathStyle  bool
	Timeout         time.Duration
}
