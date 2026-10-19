package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
)

// Shared helpers for the RES-022 media tests.
//
// These files are deliberately NOT behind the `integration` build tag: the
// handlers package only wires config.DB inside the integration-tagged TestMain,
// so an untagged run has no database at all. Every test that needs one calls
// skipWithoutDB first, which keeps `go test ./handlers/` green on a laptop and
// `go test -tags integration ./handlers/` meaningful in CI.

// skipWithoutDB skips a test that needs the integration database.
func skipWithoutDB(t *testing.T) {
	t.Helper()
	if config.DB == nil {
		t.Skip("requires the integration test database (run with -tags integration)")
	}
}

// mediaStore is a local-disk file store rooted in a temp directory.
//
// The media backend is content-addressed and content-type agnostic on disk, so
// writing the bytes directly is enough to make a key "exist" for the attach and
// serve paths — the same thing a client PUT to the presigned URL would do.
type mediaStore struct {
	t    *testing.T
	root string
}

// newMediaStore points the storage service at a fresh temp directory and
// forces the local-disk backend, so a machine (or CI job) with R2 configured
// cannot send these tests to the network.
func newMediaStore(t *testing.T) *mediaStore {
	t.Helper()
	root := t.TempDir()
	t.Setenv("STORAGE_ROOT", root)
	t.Setenv("MEDIA_ENDPOINT", "")
	t.Setenv("MEDIA_ACCESS_KEY_ID", "")
	t.Setenv("MEDIA_SECRET_ACCESS_KEY", "")
	t.Setenv("MEDIA_BUCKET", "")
	return &mediaStore{t: t, root: root}
}

// put writes body under <category>/<yyyy>/<mm>/<sha256>.<ext> — the key layout
// the presign endpoint hands out — and returns the key and the bare hash.
func (s *mediaStore) put(category, filename string, body []byte) (key string, hash string) {
	s.t.Helper()
	sum := sha256.Sum256(body)
	hash = hex.EncodeToString(sum[:])
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".bin"
	}
	now := time.Now().UTC()
	key = path.Join(category, now.Format("2006"), now.Format("01"), hash+ext)

	full := filepath.Join(s.root, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		s.t.Fatalf("create media dir: %v", err)
	}
	if err := os.WriteFile(full, body, 0o644); err != nil {
		s.t.Fatalf("write media object: %v", err)
	}
	return key, hash
}

// exists reports whether the object is still on disk.
func (s *mediaStore) exists(key string) bool {
	s.t.Helper()
	_, err := os.Stat(filepath.Join(s.root, filepath.FromSlash(key)))
	return err == nil
}

// read returns the stored bytes of an object.
func (s *mediaStore) read(key string) []byte {
	s.t.Helper()
	body, err := os.ReadFile(filepath.Join(s.root, filepath.FromSlash(key)))
	if err != nil {
		s.t.Fatalf("read media object %s: %v", key, err)
	}
	return body
}

// mediaContext builds the gin.Context routes.go hands to a handler: the caller
// in the "user" key, the route params set, and a JSON body.
func mediaContext(t *testing.T, method, target string, params gin.Params, user *models.User, body any) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	var reader io.Reader = http.NoBody
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, reader)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = params
	if user != nil {
		ctx.Set("user", user)
	}
	return ctx, recorder
}

// mediaAssignUserToUnit persists user.unit_id. Several authorization
// predicates read the column from the database, so the field must be stored
// and not only set on the in-memory struct.
func mediaAssignUserToUnit(t *testing.T, user *models.User, unit *models.SecurityUnit) {
	t.Helper()
	user.UnitID = &unit.ID
	if err := config.DB.Model(&models.User{}).Where("id = ?", user.ID).
		Update("unit_id", unit.ID).Error; err != nil {
		t.Fatalf("assign unit: %v", err)
	}
}

// mediaLinkCount counts the media_links rows for one object key. A nil
// parentID counts every parent that links to the key.
func mediaLinkCount(t *testing.T, objectKey string, parentID *uuid.UUID) int64 {
	t.Helper()
	q := config.DB.Model(&models.MediaLink{}).Where("object_key = ?", objectKey)
	if parentID != nil {
		q = q.Where("parent_id = ?", *parentID)
	}
	var n int64
	if err := q.Count(&n).Error; err != nil {
		t.Fatalf("count media links for %s: %v", objectKey, err)
	}
	return n
}

// mediaLinkID returns the id of the link row for one object key and parent.
func mediaLinkID(t *testing.T, objectKey string, parentID uuid.UUID) uuid.UUID {
	t.Helper()
	var link models.MediaLink
	if err := config.DB.Where("object_key = ? AND parent_id = ?", objectKey, parentID).
		First(&link).Error; err != nil {
		t.Fatalf("load media link for %s: %v", objectKey, err)
	}
	return link.ID
}

// mediaMultipartFile builds the *multipart.FileHeader the storage service
// expects from an upload.
func mediaMultipartFile(t *testing.T, filename, contentType string, body []byte) *multipart.FileHeader {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	header := make(map[string][]string)
	header["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="file"; filename=%q`, filename)}
	header["Content-Type"] = []string{contentType}
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("create form part: %v", err)
	}
	if _, err := part.Write(body); err != nil {
		t.Fatalf("write form part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	form, err := multipart.NewReader(&buf, writer.Boundary()).ReadForm(1 << 20)
	if err != nil {
		t.Fatalf("read multipart form: %v", err)
	}
	files := form.File["file"]
	if len(files) != 1 {
		t.Fatalf("want 1 uploaded file, got %d", len(files))
	}
	return files[0]
}
