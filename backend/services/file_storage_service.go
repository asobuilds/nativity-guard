package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type FileStorageService struct {
	Root string

	// r2 is non-nil when the R2_* environment is configured. When nil the
	// service keeps using the legacy local disk backend.
	r2    *R2Service
	r2Err error
	once  sync.Once
}

// NewFileStorageService returns a storage service backed by Cloudflare R2 when
// R2_ACCOUNT_ID and friends are set, otherwise by the local disk. The key layout
// is identical in both modes: <category>/<yyyy>/<mm>/<hash>.<ext>.
func NewFileStorageService() *FileStorageService {
	root := os.Getenv("STORAGE_ROOT")
	if root == "" {
		root = "./storage"
	}
	return &FileStorageService{Root: root}
}

// remote lazily builds the R2 client so a misconfigured production environment
// surfaces on first use instead of at process start.
func (s *FileStorageService) remote() *R2Service {
	s.once.Do(func() {
		if !R2Configured() {
			return
		}
		svc, err := NewR2Service()
		if err != nil {
			s.r2Err = err
			return
		}
		s.r2 = svc
	})
	return s.r2
}

// UsingR2 reports whether this service is backed by Cloudflare R2.
func (s *FileStorageService) UsingR2() bool {
	return s.remote() != nil
}

// R2 returns the underlying R2 client, or nil in local disk mode.
func (s *FileStorageService) R2() *R2Service {
	return s.remote()
}

// StoredFile describes what was written.
type StoredFile struct {
	RelativePath string
	Hash         string
	Size         int64
	ContentType  string
}

// allowedCategoryDirs maps a logical category to a storage subfolder.
var allowedCategoryDirs = map[string]string{
	"avatar":     "avatars",
	"cover":      "covers",
	"evidence":   "evidence",
	"gov_id":     "gov_ids",
	"voice_note": "voice_notes",
	"unit_logo":  "unit_logos",
	"unit_cover": "unit_covers",
}

// Save persists the uploaded file under <category-dir>/<yyyy>/<mm>/<hash>.<ext>
// Content-addressed: identical files deduplicate. Save keeps the same signature in
// both backends, so callers do not change when R2 is enabled.
func (s *FileStorageService) Save(file *multipart.FileHeader, category string) (*StoredFile, error) {
	if file == nil {
		return nil, errors.New("no file provided")
	}

	dirName, ok := allowedCategoryDirs[category]
	if !ok {
		return nil, errors.New("invalid storage category")
	}

	src, err := file.Open()
	if err != nil {
		return nil, errors.New("cannot open file")
	}
	defer src.Close()

	hasher := sha256.New()
	// Buffer in memory: R2 needs the full body for a single PutObject, and the
	// largest accepted upload (evidence) is capped at 100 MB by validation.
	var buf bytes.Buffer
	size, err := io.Copy(io.MultiWriter(&buf, hasher), src)
	if err != nil {
		return nil, err
	}

	hash := hex.EncodeToString(hasher.Sum(nil))
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		ext = ".bin"
	}
	contentType := file.Header.Get("Content-Type")

	if r2 := s.remote(); r2 != nil {
		key := storageKey(dirName, hash, ext)
		if _, err := r2.Upload(context.Background(), key, buf.Bytes(), contentType, map[string]string{
			"sha256": hash,
		}); err != nil {
			return nil, err
		}
		return &StoredFile{
			RelativePath: key,
			Hash:         hash,
			Size:         size,
			ContentType:  contentType,
		}, nil
	}

	stored, err := s.saveLocal(dirName, hash, ext, contentType, &buf, size)
	if err != nil {
		return nil, err
	}
	return stored, nil
}

// storageKey builds the shared <category>/<yyyy>/<mm>/<hash>.<ext> key.
func storageKey(dirName, hash, ext string) string {
	now := time.Now().UTC()
	return filepath.ToSlash(filepath.Join(dirName, now.Format("2006"), now.Format("01"), hash+ext))
}

// saveLocal is the unchanged legacy local disk write path.
func (s *FileStorageService) saveLocal(dirName, hash, ext, contentType string, buf *bytes.Buffer, size int64) (*StoredFile, error) {
	now := time.Now().UTC()
	tmpDir := filepath.Join(s.Root, dirName, now.Format("2006"), now.Format("01"))
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return nil, err
	}

	tmpFile, err := os.CreateTemp(tmpDir, "upload-*.tmp")
	if err != nil {
		return nil, err
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	if _, err := io.Copy(tmpFile, buf); err != nil {
		tmpFile.Close()
		return nil, err
	}
	if err := tmpFile.Close(); err != nil {
		return nil, err
	}

	finalName := hash + ext
	finalPath := filepath.Join(tmpDir, finalName)

	// Deduplicate: if the same hash already exists, discard the temp and reuse.
	if _, err := os.Stat(finalPath); err == nil {
		_ = os.Remove(tmpName)
	} else {
		if err := os.Rename(tmpName, finalPath); err != nil {
			return nil, err
		}
	}

	rel, err := filepath.Rel(s.Root, finalPath)
	if err != nil {
		rel = finalPath
	}

	return &StoredFile{
		RelativePath: filepath.ToSlash(rel),
		Hash:         hash,
		Size:         size,
		ContentType:  contentType,
	}, nil
}

// Open returns a reader for a stored key. Caller must close.
func (s *FileStorageService) Open(relativePath string) (io.ReadCloser, error) {
	clean := sanitizeKey(relativePath)
	if clean == "" {
		return nil, errors.New("invalid path")
	}

	if r2 := s.remote(); r2 != nil {
		body, err := r2.Open(context.Background(), clean)
		if err != nil {
			return nil, err
		}
		return body, nil
	}

	full := filepath.Join(s.Root, filepath.FromSlash(clean))
	return os.Open(full)
}

// Delete removes a stored file. Missing files are not an error.
func (s *FileStorageService) Delete(relativePath string) error {
	clean := sanitizeKey(relativePath)
	if clean == "" {
		return errors.New("invalid path")
	}

	if r2 := s.remote(); r2 != nil {
		return r2.Delete(context.Background(), clean)
	}

	full := filepath.Join(s.Root, filepath.FromSlash(clean))
	err := os.Remove(full)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Exists reports whether a stored file is present.
func (s *FileStorageService) Exists(relativePath string) bool {
	clean := sanitizeKey(relativePath)
	if clean == "" {
		return false
	}

	if r2 := s.remote(); r2 != nil {
		return r2.Exists(context.Background(), clean)
	}

	full := filepath.Join(s.Root, filepath.FromSlash(clean))
	_, err := os.Stat(full)
	return err == nil
}

// FindByHash resolves a category plus bare hash to a full key, walking
// <category>/<yyyy>/<mm>/<hash>.<ext>. Returns "" when nothing matches.
func (s *FileStorageService) FindByHash(category string, hash string) (string, error) {
	cleanCategory := sanitizeKey(category)
	cleanHash := strings.ToLower(strings.TrimSpace(hash))
	if cleanCategory == "" || cleanHash == "" {
		return "", nil
	}

	if r2 := s.remote(); r2 != nil {
		return r2.FindByHash(context.Background(), cleanCategory, cleanHash)
	}

	matches, err := filepath.Glob(filepath.Join(s.Root, cleanCategory, "*", "*", cleanHash+".*"))
	if err != nil || len(matches) == 0 {
		return "", nil
	}
	sortStrings(matches)

	rel, err := filepath.Rel(s.Root, matches[0])
	if err != nil {
		rel = matches[0]
	}
	return filepath.ToSlash(rel), nil
}

// sortStrings is a tiny insertion sort to keep the import surface minimal.
func sortStrings(items []string) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j] < items[j-1]; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

// sanitizeKey rejects traversal and returns a slash-normalised key, or "" when
// the input is unsafe.
func sanitizeKey(p string) string {
	clean := filepath.ToSlash(filepath.Clean(strings.ReplaceAll(p, "\\", "/")))
	if clean == "" || strings.HasPrefix(clean, "..") || strings.Contains(clean, "../") {
		return ""
	}
	if clean == "." || clean == ".." {
		return ""
	}
	return clean
}
