package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// R2Service talks to a Supabase Storage bucket through the S3-compatible API.
//
// Bucket policy is split by key prefix:
//   - public  : avatars/, covers/, units/, news/, community/  -> served via MEDIA_PUBLIC_BASE
//   - private : evidence/, gov_ids/, voice_notes/           -> never exposed as a public URL
//
// Private objects are only reachable through an authenticated handler that
// redirects to a short-lived presigned GET URL.
type R2Service struct {
	client     *s3.Client
	presigner  *s3.PresignClient
	bucket     string
	publicBase string
}

// publicKeyPrefixes are the key prefixes that may be exposed on the CDN base.
var publicKeyPrefixes = []string{
	"avatars/",
	"covers/",
	"units/",
	"news/",
	"community/",
}

// privateKeyPrefixes require authenticated access. They are never linked publicly.
var privateKeyPrefixes = []string{
	"evidence/",
	"gov_ids/",
	"voice_notes/",
	"sos_media/",
	"report_media/",
}

// NewR2Service builds a client from the MEDIA_* environment variables.
//
//	MEDIA_ENDPOINT        S3 endpoint, e.g. https://<project>.supabase.co/storage/v1/s3
//	MEDIA_ACCESS_KEY_ID   S3 access key id
//	MEDIA_SECRET_ACCESS_KEY  S3 secret access key
//	MEDIA_BUCKET          bucket name
//	MEDIA_PUBLIC_BASE     public CDN base, e.g. https://<project>.supabase.co/storage/v1/object/public/<bucket>
//	MEDIA_REGION          AWS region (default: us-east-1)
func NewR2Service() (*R2Service, error) {
	endpoint := strings.TrimSpace(os.Getenv("MEDIA_ENDPOINT"))
	accessKey := strings.TrimSpace(os.Getenv("MEDIA_ACCESS_KEY_ID"))
	secretKey := strings.TrimSpace(os.Getenv("MEDIA_SECRET_ACCESS_KEY"))
	bucket := strings.TrimSpace(os.Getenv("MEDIA_BUCKET"))
	publicBase := strings.TrimRight(strings.TrimSpace(os.Getenv("MEDIA_PUBLIC_BASE")), "/")
	region := resolveS3Region(endpoint, os.Getenv("MEDIA_REGION"))

	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		return nil, fmt.Errorf("media: MEDIA_ENDPOINT, MEDIA_ACCESS_KEY_ID, MEDIA_SECRET_ACCESS_KEY and MEDIA_BUCKET are required")
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("media: load config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	})

	return &R2Service{
		client:     client,
		presigner:  s3.NewPresignClient(client),
		bucket:     bucket,
		publicBase: publicBase,
	}, nil
}


// resolveS3Region uses Cloudflare R2's SigV4 region, regardless of the
// bucket's physical location or an AWS-style region saved in old deployments.
// For non-R2 endpoints (including Supabase S3), preserve the configured region.
func resolveS3Region(endpoint, configured string) string {
  parsed, err := url.Parse(strings.TrimSpace(endpoint))
  if err == nil && strings.HasSuffix(strings.ToLower(parsed.Hostname()), ".r2.cloudflarestorage.com") {
    return "auto"
  }
  region := strings.TrimSpace(configured)
  if region == "" { return "us-east-1" }
  return region
}

// R2Configured reports whether the MEDIA_* environment is present. When false the
// storage layer keeps using the legacy local disk backend.
func R2Configured() bool {
	return strings.TrimSpace(os.Getenv("MEDIA_ENDPOINT")) != "" &&
		strings.TrimSpace(os.Getenv("MEDIA_ACCESS_KEY_ID")) != "" &&
		strings.TrimSpace(os.Getenv("MEDIA_SECRET_ACCESS_KEY")) != "" &&
		strings.TrimSpace(os.Getenv("MEDIA_BUCKET")) != ""
}

// IsPublicKey reports whether a key belongs to a public prefix. Unknown prefixes
// are treated as private so a new category can never leak by accident.
func (s *R2Service) IsPublicKey(key string) bool {
	return IsPublicStorageKey(key)
}

// IsPublicStorageKey is the package-level form of R2Service.IsPublicKey.
func IsPublicStorageKey(key string) bool {
	k := strings.TrimPrefix(strings.ReplaceAll(key, "\\", "/"), "./")
	for _, p := range publicKeyPrefixes {
		if strings.HasPrefix(k, p) {
			return true
		}
	}
	return false
}

// IsPrivateStorageKey reports whether a key belongs to a private prefix.
func IsPrivateStorageKey(key string) bool {
	k := strings.TrimPrefix(strings.ReplaceAll(key, "\\", "/"), "./")
	for _, p := range privateKeyPrefixes {
		if strings.HasPrefix(k, p) {
			return true
		}
	}
	return false
}

// PublicURL returns the CDN URL for a key. It is only valid for public keys;
// callers must check IsPublicKey first and fall back to PresignGet otherwise.
func (s *R2Service) PublicURL(key string) string {
	if s.publicBase == "" {
		return ""
	}
	return s.publicBase + "/" + strings.TrimPrefix(key, "/")
}

// Upload stores bytes at key and returns the public URL for public keys, or the
// key itself for private keys so a public URL is never handed out for evidence.
func (s *R2Service) Upload(ctx context.Context, key string, body []byte, contentType string, metadata map[string]string) (string, error) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String(contentType),
	}
	if len(metadata) > 0 {
		input.Metadata = normalizeMetadata(metadata)
	}

	if _, err := s.client.PutObject(ctx, input); err != nil {
		return "", fmt.Errorf("media: put %s: %w", key, err)
	}

	if IsPublicStorageKey(key) {
		return s.PublicURL(key), nil
	}
	return key, nil
}

// PresignPut returns a short-lived URL the client can PUT to directly, so large
// uploads travel from the browser straight to R2 instead of through this server.
func (s *R2Service) PresignPut(ctx context.Context, key string, contentType string, sizeBytes int64, metadata map[string]string, ttl time.Duration) (string, error) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}
	if sizeBytes > 0 {
		input.ContentLength = aws.Int64(sizeBytes)
	}
	if len(metadata) > 0 {
		input.Metadata = normalizeMetadata(metadata)
	}

	req, err := s.presigner.PresignPutObject(ctx, input, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("media: presign put %s: %w", key, err)
	}
	return fixPresignedURL(req.URL)
}

// PresignGet returns a short-lived read URL. Used for private prefixes only.
func (s *R2Service) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}

	req, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("media: presign get %s: %w", key, err)
	}
	return fixPresignedURL(req.URL)
}

// Head returns the stored size and content type for a key.
func (s *R2Service) Head(ctx context.Context, key string) (int64, string, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return 0, "", fmt.Errorf("media: head %s: %w", key, err)
	}

	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	ct := ""
	if out.ContentType != nil {
		ct = *out.ContentType
	}
	return size, ct, nil
}

// Open returns a reader for a stored key. Caller must close.
func (s *R2Service) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("media: get %s: %w", key, err)
	}
	return out.Body, nil
}

// Delete removes an object by key. Missing objects are not an error.
func (s *R2Service) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err == nil || isNotFound(err) {
		return nil
	}
	return fmt.Errorf("media: delete %s: %w", key, err)
}

// Exists reports whether an object is present.
func (s *R2Service) Exists(ctx context.Context, key string) bool {
	_, _, err := s.Head(ctx, key)
	return err == nil
}

// FindByHash locates the object key for a category and bare hash by walking
// <category>/<yyyy>/<mm>/<hash>.<ext>. It returns "" when nothing matches.
func (s *R2Service) FindByHash(ctx context.Context, category string, hash string) (string, error) {
	if hash == "" || strings.Contains(hash, "..") || strings.ContainsAny(hash, `/\`) {
		return "", nil
	}

	prefix := strings.Trim(category, "/") + "/"
	continuation := aws.String("")

	for {
		page, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(s.bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: continuation,
		})
		if err != nil {
			return "", fmt.Errorf("media: list %s: %w", prefix, err)
		}

		for _, obj := range page.Contents {
			k := aws.ToString(obj.Key)
			if keyMatchesHash(k, category, hash) {
				return k, nil
			}
		}

		if page.IsTruncated == nil || !*page.IsTruncated {
			break
		}
		continuation = page.NextContinuationToken
		if continuation == nil {
			break
		}
	}

	return "", nil
}

// keyMatchesHash reports whether a full object key is the <yyyy>/<mm>/<hash>.<ext>
// layout for the given category and matches the bare hash.
func keyMatchesHash(key string, category string, hash string) bool {
	k := strings.TrimPrefix(key, "/")
	prefix := strings.Trim(category, "/") + "/"
	if !strings.HasPrefix(k, prefix) {
		return false
	}

	rel := strings.TrimPrefix(k, prefix)
	parts := strings.Split(rel, "/")
	// yyyy / mm / <hash>.<ext>
	if len(parts) < 3 {
		return false
	}
	name := parts[len(parts)-1]
	base := strings.TrimSuffix(name, pathExt(name))
	return base == hash
}

func pathExt(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[i:]
	}
	return ""
}

// fixPresignedURL only validates the generated URL. Mutating ANY SigV4 query
// parameter (including credential scope/region) after signing makes the URL's
// authentication signature invalid. The signer must receive the right region
// through resolveS3Region before it signs the request.
func fixPresignedURL(raw string) (string, error) {
  parsed, err := url.Parse(raw)
  if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
    return "", fmt.Errorf("r2: invalid presigned URL")
  }
  return raw, nil
}

func normalizeMetadata(metadata map[string]string) map[string]string {
	out := make(map[string]string, len(metadata))
	for k, v := range metadata {
		out[strings.ToLower(strings.TrimSpace(k))] = v
	}
	return out
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	var nsk *types.NoSuchKey
	if errors.As(err, &nsk) {
		return true
	}
	var nf *types.NotFound
	if errors.As(err, &nf) {
		return true
	}
	return false
}
