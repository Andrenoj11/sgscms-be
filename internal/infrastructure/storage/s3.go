package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3 struct {
	client            *minio.Client
	bucket, publicURL string
}

func NewS3(ctx context.Context, endpoint, bucket, access, secret, region, publicURL string, ssl bool) (*S3, error) {
	client, err := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4(access, secret, ""), Secure: ssl, Region: region})
	if err != nil {
		return nil, err
	}
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: region}); err != nil {
			return nil, err
		}
	}
	if publicURL == "" {
		scheme := "http"
		if ssl {
			scheme = "https"
		}
		publicURL = fmt.Sprintf("%s://%s/%s", scheme, endpoint, bucket)
	}
	return &S3{client: client, bucket: bucket, publicURL: strings.TrimRight(publicURL, "/")}, nil
}
func (s *S3) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", err
	}
	return s.publicURL + "/" + key, nil
}
func (s *S3) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}
