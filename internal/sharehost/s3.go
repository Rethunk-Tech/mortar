package sharehost

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Store is the DigitalOcean Spaces (S3-compatible) store.
// Endpoint, bucket, key and secret come from env; they are never logged.
type S3Store struct {
	client *minio.Client
	bucket string
}

var _ Store = (*S3Store)(nil)

func NewS3Store(endpoint, bucket, accessKey, secretKey string) (*S3Store, error) {
	endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: true,
	})
	if err != nil {
		return nil, err
	}
	return &S3Store{client: client, bucket: bucket}, nil
}

func shareCodeFromObjectKey(key string) (string, bool) {
	if !strings.HasSuffix(key, "/meta.json") {
		return "", false
	}
	code := strings.TrimSuffix(strings.TrimPrefix(key, "shares/"), "/meta.json")
	if code == "" || strings.Contains(code, "/") {
		return "", false
	}
	return code, true
}

func metaKey(code string) string { return "shares/" + code + "/meta.json" }

func verKey(code string, n int) string {
	return "shares/" + code + "/" + strconv.Itoa(n) + ".json"
}

func (s *S3Store) Create(ctx context.Context, code string, meta Meta, ver Version) error {
	if _, err := s.get(ctx, metaKey(code)); err == nil {
		return ErrExists
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	if err := s.putJSON(ctx, verKey(code, meta.Latest), ver); err != nil {
		return err
	}
	return s.putJSON(ctx, metaKey(code), meta)
}

func (s *S3Store) Meta(ctx context.Context, code string) (Meta, error) {
	raw, err := s.get(ctx, metaKey(code))
	if err != nil {
		return Meta{}, err
	}
	var m Meta
	if err := json.Unmarshal(raw, &m); err != nil {
		return Meta{}, err
	}
	return m, nil
}

func (s *S3Store) SaveMeta(ctx context.Context, code string, meta Meta) error {
	if _, err := s.Meta(ctx, code); err != nil {
		return err
	}
	return s.putJSON(ctx, metaKey(code), meta)
}

func (s *S3Store) SaveVersion(ctx context.Context, code string, n int, ver Version) error {
	if _, err := s.Meta(ctx, code); err != nil {
		return err
	}
	return s.putJSON(ctx, verKey(code, n), ver)
}

func (s *S3Store) LoadVersion(ctx context.Context, code string, n int) (Version, error) {
	raw, err := s.get(ctx, verKey(code, n))
	if err != nil {
		return Version{}, err
	}
	var v Version
	if err := json.Unmarshal(raw, &v); err != nil {
		return Version{}, err
	}
	return v, nil
}

func (s *S3Store) Sweep(ctx context.Context, cutoff time.Time) (deleted, kept int, err error) {
	codes := map[string]struct{}{}
	for obj := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    "shares/",
		Recursive: true,
	}) {
		if obj.Err != nil {
			return deleted, kept, obj.Err
		}
		code, ok := shareCodeFromObjectKey(obj.Key)
		if !ok {
			continue
		}
		codes[code] = struct{}{}
	}
	for code := range codes {
		meta, err := s.Meta(ctx, code)
		if err != nil {
			return deleted, kept, err
		}
		if !ageRef(meta).After(cutoff) {
			if err := s.deleteCode(ctx, code); err != nil {
				return deleted, kept, err
			}
			deleted++
			continue
		}
		kept++
	}
	return deleted, kept, nil
}

func (s *S3Store) deleteCode(ctx context.Context, code string) error {
	prefix := "shares/" + code + "/"
	for obj := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}) {
		if obj.Err != nil {
			return obj.Err
		}
		if err := s.client.RemoveObject(ctx, s.bucket, obj.Key, minio.RemoveObjectOptions{}); err != nil {
			return err
		}
	}
	return nil
}

func (s *S3Store) putJSON(ctx context.Context, key string, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(raw), int64(len(raw)), minio.PutObjectOptions{
		ContentType: "application/json",
	})
	return err
}

func (s *S3Store) get(ctx context.Context, key string) ([]byte, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		if isNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	defer func() { _ = obj.Close() }()
	raw, err := io.ReadAll(obj)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return raw, nil
}

func isNotFound(err error) bool {
	resp := minio.ToErrorResponse(err)
	return resp.StatusCode == http.StatusNotFound || resp.Code == "NoSuchKey" || resp.Code == "NoSuchObject"
}
