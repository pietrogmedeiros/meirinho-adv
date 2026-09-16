// Package storage guarda o áudio das audiências no MinIO (API S3).
//
// Áudio de audiência é dado sensível (LGPD): o upload pede criptografia em
// repouso quando MINIO_SSE_ENABLED está ligado, e o acesso ao arquivo é sempre
// por URL pré-assinada de vida curta — nunca por bucket público.
package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/encrypt"
)

type Client struct {
	mc     *minio.Client
	bucket string
	sse    encrypt.ServerSide
}

type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	SSE       bool
}

func New(ctx context.Context, cfg Config) (*Client, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("cliente minio: %w", err)
	}

	c := &Client{mc: mc, bucket: cfg.Bucket}
	if cfg.SSE {
		c.sse = encrypt.NewSSE()
	}
	return c, nil
}

// GarantirBucket cria o bucket no boot se ele ainda não existir.
func (c *Client) GarantirBucket(ctx context.Context, tentativas int) error {
	var last error
	for i := 0; i < tentativas; i++ {
		ok, err := c.mc.BucketExists(ctx, c.bucket)
		if err == nil {
			if ok {
				return nil
			}
			if err := c.mc.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{}); err == nil {
				return nil
			} else {
				last = err
			}
		} else {
			last = err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return fmt.Errorf("minio indisponível: %w", last)
}

func (c *Client) Upload(ctx context.Context, key string, r io.Reader, tamanho int64, contentType string) error {
	_, err := c.mc.PutObject(ctx, c.bucket, key, r, tamanho, minio.PutObjectOptions{
		ContentType:          contentType,
		ServerSideEncryption: c.sse,
	})
	if err != nil {
		return fmt.Errorf("upload %s: %w", key, err)
	}
	return nil
}

func (c *Client) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := c.mc.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{ServerSideEncryption: c.sse})
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", key, err)
	}
	return obj, nil
}

// URLAssinada devolve um link temporário para o front tocar o áudio sem que o
// bucket precise ser público.
func (c *Client) URLAssinada(ctx context.Context, key string, validade time.Duration) (string, error) {
	u, err := c.mc.PresignedGetObject(ctx, c.bucket, key, validade, url.Values{})
	if err != nil {
		return "", fmt.Errorf("assinar url %s: %w", key, err)
	}
	return u.String(), nil
}
