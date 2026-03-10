package updater

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type HTTPDownloader struct {
	client     *http.Client
	maxRetries int
}

func NewHTTPDownloader() *HTTPDownloader {
	return &HTTPDownloader{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxRetries: 1,
	}
}

func NewHTTPDownloaderWithRetry(n int) *HTTPDownloader {
	return &HTTPDownloader{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxRetries: n,
	}
}

func (d *HTTPDownloader) Download(ctx context.Context, url, destPath string) error {
	var lastErr error

	for attempt := 0; attempt < d.maxRetries; attempt++ {
		lastErr = d.doDownload(ctx, url, destPath)
		if lastErr == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return lastErr
}

func (d *HTTPDownloader) doDownload(ctx context.Context, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: status %d", resp.StatusCode)
	}

	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (d *HTTPDownloader) DownloadWithChecksum(ctx context.Context, url, destPath, expectedChecksum string) error {
	actualChecksum := expectedChecksum

	parts := strings.Split(expectedChecksum, ":")
	if len(parts) == 2 {
		actualChecksum = parts[1]
	}

	if err := d.Download(ctx, url, destPath); err != nil {
		return err
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		return fmt.Errorf("failed to read downloaded file: %w", err)
	}

	hash := md5.Sum(data)
	checksum := hex.EncodeToString(hash[:])

	if checksum != actualChecksum && expectedChecksum != checksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, checksum)
	}

	return nil
}
