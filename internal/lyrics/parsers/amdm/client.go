package amdm

import (
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sukalov/karaokebot/internal/logger"
)

// Client represents the HTTP client for AmDm requests
type Client struct {
	httpClient *http.Client
	userAgent  string
}

// NewClient creates a new AmDm HTTP client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
					MinVersion:         tls.VersionTLS12,
					MaxVersion:         tls.VersionTLS13,
				},
				DisableCompression: false,
			},
		},
		userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	}
}

// FetchPage fetches the HTML content from the given URL
func (c *Client) FetchPage(url string) (string, error) {
	fetchURL := strings.Replace(url, "123.amdm.ru", "amdm.ru", 1)
	req, err := http.NewRequest("GET", fetchURL, nil)
	if err != nil {
		logger.Error(false, fmt.Sprintf("Failed to create HTTP request\nURL: %s\nError: %v", fetchURL, err))
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Error(false, fmt.Sprintf("Failed to fetch page\nURL: %s\nError: %v", fetchURL, err))
		return "", fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error(false, fmt.Sprintf("HTTP error fetching page\nURL: %s\nStatus: %d", fetchURL, resp.StatusCode))
		return "", fmt.Errorf("HTTP error! status: %d", resp.StatusCode)
	}

	var reader io.Reader = resp.Body

	// Handle gzip decompression
	if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			logger.Error(false, fmt.Sprintf("Failed to create gzip reader\nURL: %s\nError: %v", fetchURL, err))
			return "", fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		logger.Error(false, fmt.Sprintf("Failed to read response body\nURL: %s\nError: %v", fetchURL, err))
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}
