package updater

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultRepo    = "itda-work/zap"
	apiBaseURL     = "https://api.github.com"
	defaultTimeout = 30 * time.Second
)

// newHTTP1Client creates an HTTP client that forces HTTP/1.1.
// This avoids HTTP/2 protocol errors with some CDNs.
func newHTTP1Client(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSNextProto: make(map[string]func(string, *tls.Conn) http.RoundTripper),
		},
	}
}

// ReleaseInfo holds information about a GitHub release.
type ReleaseInfo struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	PublishedAt time.Time `json:"published_at"`
	Prerelease  bool      `json:"prerelease"`
	Draft       bool      `json:"draft"`
	Assets      []Asset   `json:"assets"`
	HTMLURL     string    `json:"html_url"`
}

// Asset represents a release asset (downloadable file).
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	ContentType        string `json:"content_type"`
}

// GitHubClient handles GitHub API interactions.
type GitHubClient struct {
	repo       string
	httpClient *http.Client
}

// NewGitHubClient creates a new GitHub client for the default repository.
func NewGitHubClient() *GitHubClient {
	return NewGitHubClientWithRepo(defaultRepo)
}

// NewGitHubClientWithRepo creates a new GitHub client for a specific repository.
func NewGitHubClientWithRepo(repo string) *GitHubClient {
	return &GitHubClient{
		repo: repo,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// GetLatestRelease fetches the latest non-prerelease version.
func (c *GitHubClient) GetLatestRelease() (*ReleaseInfo, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", apiBaseURL, c.repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "zap-updater")

	// Use GitHub token if available
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &NetworkError{Err: err}
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp); err != nil {
		return nil, err
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &release, nil
}

// GetRelease fetches a specific release by tag.
func (c *GitHubClient) GetRelease(tag string) (*ReleaseInfo, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/tags/%s", apiBaseURL, c.repo, tag)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "zap-updater")

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &NetworkError{Err: err}
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp); err != nil {
		return nil, err
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &release, nil
}

// DownloadAsset downloads an asset to the specified path with progress reporting.
func (c *GitHubClient) DownloadAsset(asset *Asset, destPath string, progress func(downloaded, total int64)) error {
	req, err := http.NewRequest("GET", asset.BrowserDownloadURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "zap-updater")

	// Use HTTP/1.1 client for downloads to avoid HTTP/2 protocol errors with CDNs
	downloadClient := newHTTP1Client(defaultTimeout)
	resp, err := downloadClient.Do(req)
	if err != nil {
		return &NetworkError{Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	total := resp.ContentLength
	var downloaded int64

	var reader io.Reader = resp.Body
	if progress != nil {
		reader = &progressReader{
			reader: resp.Body,
			onProgress: func(n int64) {
				downloaded += n
				progress(downloaded, total)
			},
		}
	}

	if _, err := io.Copy(out, reader); err != nil {
		return fmt.Errorf("download: %w", err)
	}

	return nil
}

// GetChecksums downloads and parses checksums.txt from the release.
func (c *GitHubClient) GetChecksums(release *ReleaseInfo) (map[string]string, error) {
	var checksumAsset *Asset
	for i := range release.Assets {
		if release.Assets[i].Name == "checksums.txt" {
			checksumAsset = &release.Assets[i]
			break
		}
	}

	if checksumAsset == nil {
		return nil, fmt.Errorf("checksums.txt not found in release")
	}

	req, err := http.NewRequest("GET", checksumAsset.BrowserDownloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "zap-updater")

	// Use HTTP/1.1 client for downloads to avoid HTTP/2 protocol errors with CDNs
	downloadClient := newHTTP1Client(defaultTimeout)
	resp, err := downloadClient.Do(req)
	if err != nil {
		return nil, &NetworkError{Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download checksums failed with status: %s", resp.Status)
	}

	checksums := make(map[string]string)
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 2 {
			// Format: "checksum  filename"
			checksums[parts[1]] = parts[0]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse checksums: %w", err)
	}

	return checksums, nil
}

// checkResponse checks for API errors in the response.
func (c *GitHubClient) checkResponse(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return &NotFoundError{Message: "release not found"}
	case http.StatusForbidden:
		// Check if it's rate limiting
		if resp.Header.Get("X-RateLimit-Remaining") == "0" {
			resetTime := resp.Header.Get("X-RateLimit-Reset")
			return &RateLimitError{ResetTime: resetTime}
		}
		return fmt.Errorf("access forbidden")
	default:
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}
}

// progressReader wraps an io.Reader to report progress.
type progressReader struct {
	reader     io.Reader
	onProgress func(n int64)
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 && r.onProgress != nil {
		r.onProgress(int64(n))
	}
	return n, err
}

// Error types for specific error handling

// NetworkError represents a network-related error.
type NetworkError struct {
	Err error
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("network error: %v", e.Err)
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}

// NotFoundError represents a 404 response.
type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string {
	return e.Message
}

// RateLimitError represents API rate limiting.
type RateLimitError struct {
	ResetTime string
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded, resets at %s", e.ResetTime)
}
