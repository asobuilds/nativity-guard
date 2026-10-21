package services

import (
  "context"
  "net/url"
  "strings"
  "testing"
  "time"
)

func TestResolveS3RegionForCloudflareR2(t *testing.T) {
  endpoint := "https://exampleaccount.r2.cloudflarestorage.com"
  for _, configured := range []string{"", "eu-west-2", "us-east-1", "auto"} {
    if got := resolveS3Region(endpoint, configured); got != "auto" {
      t.Fatalf("Cloudflare R2 configured=%q: expected auto, got %q", configured, got)
    }
  }
  if got := resolveS3Region("https://example.supabase.co/storage/v1/s3", "eu-west-2"); got != "eu-west-2" {
    t.Fatalf("non-R2 region must be preserved, got %q", got)
  }
}

func TestFixPresignedURLDoesNotMutateSignedValues(t *testing.T) {
  original := "https://r2.example.test/object?X-Amz-Credential=access%2F20261022%2Fauto%2Fs3%2Faws4_request&X-Amz-Signature=deadbeef&X-Amz-Date=20261022T000000Z"
  got, err := fixPresignedURL(original)
  if err != nil || got != original {
    t.Fatalf("signed URL must not be rewritten after signing: %q, %v", got, err)
  }
}

func TestR2SignerUsesCloudflareAutoScope(t *testing.T) {
  t.Setenv("MEDIA_ENDPOINT", "https://exampleaccount.r2.cloudflarestorage.com")
  t.Setenv("MEDIA_REGION", "eu-west-2")
  t.Setenv("MEDIA_ACCESS_KEY_ID", "test-access-key")
  t.Setenv("MEDIA_SECRET_ACCESS_KEY", "not-a-real-secret")
  t.Setenv("MEDIA_BUCKET", "test-bucket")
  client, err := NewR2Service()
  if err != nil { t.Fatal(err) }
  signed, err := client.PresignPut(context.Background(), "sos_media/2026/10/example.webm",
    "audio/webm", 20, map[string]string{"sha256":strings.Repeat("0",64)}, time.Minute)
  if err != nil { t.Fatal(err) }
  parsed, err := url.Parse(signed)
  if err != nil { t.Fatal(err) }
  scope := parsed.Query().Get("X-Amz-Credential")
  if !strings.Contains(scope, "/auto/s3/aws4_request") {
    t.Fatalf("signed request region must be auto, got %q", scope)
  }
  if parsed.Query().Get("X-Amz-Signature")=="" { t.Fatal("missing SigV4 signature") }
}
