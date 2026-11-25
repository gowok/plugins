package s3

import (
	"bytes"
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gowok/gowok"
)

type client struct {
	s3.Client
	bucket string
}

func (c client) Get(ctx context.Context, key string) ([]byte, error) {
	res, err := c.GetObject(ctx, &s3.GetObjectInput{
		Bucket: gowok.ToPtr(c.bucket),
		Key:    gowok.ToPtr(key),
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	out, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (c client) Put(ctx context.Context, key string, body []byte) error {
	_, err := c.PutObject(ctx, &s3.PutObjectInput{
		Bucket: gowok.ToPtr(c.bucket),
		Key:    gowok.ToPtr(key),
		Body:   bytes.NewReader(body),
	})
	if err != nil {
		return err
	}

	return nil
}
