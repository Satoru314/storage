package s3

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	s3Client     *s3.Client
	presignClient *s3.PresignClient
	bucket       string
}

func NewClient(region, bucket, accessKeyID, secretAccessKey string) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKeyID,
			secretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)
	presignClient := s3.NewPresignClient(s3Client)

	return &Client{
		s3Client:     s3Client,
		presignClient: presignClient,
		bucket:       bucket,
	}, nil
}

func (c *Client) PresignPutObject(ctx context.Context, key string, contentType string, ttl time.Duration) (string, error) {
	req, err := c.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      &c.bucket,
		Key:         &key,
		ContentType: &contentType,
	}, func(opts *s3.PresignOptions) {
		opts.Expires = ttl
	})
	if err != nil {
		return "", fmt.Errorf("failed to presign PUT object: %w", err)
	}
	return req.URL, nil
}

func (c *Client) PresignGetObject(ctx context.Context, key string, ttl time.Duration) (string, error) {
	req, err := c.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &c.bucket,
		Key:    &key,
	}, func(opts *s3.PresignOptions) {
		opts.Expires = ttl
	})
	if err != nil {
		return "", fmt.Errorf("failed to presign GET object: %w", err)
	}
	return req.URL, nil
}

func (c *Client) HeadObject(ctx context.Context, key string) (*s3.HeadObjectOutput, error) {
	output, err := c.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &c.bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to head object: %w", err)
	}
	return output, nil
}