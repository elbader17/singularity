package updater

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

type MockDownloader struct {
	downloadErr error
	content     []byte
}

func (m *MockDownloader) Download(ctx context.Context, version, destPath string) error {
	if m.downloadErr != nil {
		return m.downloadErr
	}
	return os.WriteFile(destPath, m.content, 0755)
}

func TestDownloaderDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("fake binary content"))
	}))
	defer server.Close()

	downloader := NewHTTPDownloader()
	ctx := context.Background()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "singularity")

	err := downloader.Download(ctx, server.URL, destPath)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(data) != "fake binary content" {
		t.Errorf("downloaded content = %q, want %q", string(data), "fake binary content")
	}
}

func TestDownloaderDownloadHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	downloader := NewHTTPDownloader()
	ctx := context.Background()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "singularity")

	err := downloader.Download(ctx, server.URL, destPath)
	if err == nil {
		t.Error("Download() expected error for 404, got nil")
	}
}

func TestDownloaderDownloadCreatesDirectory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("content"))
	}))
	defer server.Close()

	downloader := NewHTTPDownloader()
	ctx := context.Background()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "subdir", "singularity")

	err := downloader.Download(ctx, server.URL, destPath)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Error("Download() did not create file in nested directory")
	}
}

func TestDownloaderRetryOnFailure(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte("success"))
	}))
	defer server.Close()

	downloader := NewHTTPDownloaderWithRetry(3)
	ctx := context.Background()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "singularity")

	err := downloader.Download(ctx, server.URL, destPath)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestDownloaderVerifyChecksum(t *testing.T) {
	content := []byte("test content")
	checksum := "9473fdd0d880a43c21b7778d34872157"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer server.Close()

	downloader := NewHTTPDownloader()
	ctx := context.Background()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "singularity")

	err := downloader.DownloadWithChecksum(ctx, server.URL, destPath, checksum)
	if err != nil {
		t.Fatalf("DownloadWithChecksum() error = %v", err)
	}
}

func TestDownloaderChecksumMismatch(t *testing.T) {
	content := []byte("test content")
	checksum := "wrongchecksum"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer server.Close()

	downloader := NewHTTPDownloader()
	ctx := context.Background()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "singularity")

	err := downloader.DownloadWithChecksum(ctx, server.URL, destPath, checksum)
	if err == nil {
		t.Error("DownloadWithChecksum() expected error for checksum mismatch, got nil")
	}
}

func TestDownloaderConcurrentDownloads(t *testing.T) {
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		fmt.Fprint(w, "content")
	}))
	defer server.Close()

	downloader := NewHTTPDownloader()
	ctx := context.Background()

	tmpDir := t.TempDir()

	errs := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func(i int) {
			destPath := filepath.Join(tmpDir, fmt.Sprintf("file%d", i))
			errs <- downloader.Download(ctx, server.URL, destPath)
		}(i)
	}

	for i := 0; i < 3; i++ {
		if err := <-errs; err != nil {
			t.Errorf("Download() error = %v", err)
		}
	}
}
