package infra

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"ltc-system/apps/api/internal/modules/reporting/app"
)

// SupabaseObjectStorage 透過 Supabase Storage service-role API 存取 private bucket。
// service role key 只存在 API runtime，不會傳給瀏覽器；API 會在授權後讀回位元組並串流給前端。
type SupabaseObjectStorage struct {
	baseURL    string
	serviceKey string
	bucket     string
	client     *http.Client
}

// NewSupabaseObjectStorage 建立 Supabase Storage adapter。
func NewSupabaseObjectStorage(baseURL, serviceKey, bucket string, client *http.Client) *SupabaseObjectStorage {
	if client == nil {
		client = &http.Client{}
	}
	return &SupabaseObjectStorage{
		baseURL:    strings.TrimRight(baseURL, "/"),
		serviceKey: serviceKey,
		bucket:     strings.TrimSpace(bucket),
		client:     client,
	}
}

// Configured 回報是否具備 private object storage 所需的設定。
func (s *SupabaseObjectStorage) Configured() bool {
	return s != nil && s.baseURL != "" && s.serviceKey != "" && s.bucket != ""
}

func (s *SupabaseObjectStorage) objectURL(path string) (string, error) {
	if !s.Configured() {
		return "", fmt.Errorf("supabase object storage is not configured")
	}
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) == 0 || segments[0] == "" {
		return "", fmt.Errorf("object storage path is empty")
	}
	escaped := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return "", fmt.Errorf("object storage path contains an invalid segment")
		}
		escaped = append(escaped, url.PathEscape(segment))
	}
	return fmt.Sprintf("%s/storage/v1/object/%s/%s", s.baseURL, url.PathEscape(s.bucket), strings.Join(escaped, "/")), nil
}

func (s *SupabaseObjectStorage) request(ctx context.Context, method, path string, body io.Reader, contentType string) (*http.Response, error) {
	objectURL, err := s.objectURL(path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, objectURL, body)
	if err != nil {
		return nil, fmt.Errorf("build object storage request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("apikey", s.serviceKey)
	if method == http.MethodPost {
		// Supabase Storage 預設可能允許 upsert；明確關閉覆寫，讓 export snapshot 保持不可變。
		req.Header.Set("x-upsert", "false")
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("object storage request failed: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("object storage returned HTTP %d", resp.StatusCode)
	}
	return resp, nil
}

// Put 將檔案寫入 private bucket；同一路徑禁止覆蓋，確保 export snapshot immutable。
func (s *SupabaseObjectStorage) Put(ctx context.Context, path, contentType string, content []byte) error {
	req, err := s.request(ctx, http.MethodPost, path, bytes.NewReader(content), contentType)
	if err != nil {
		return err
	}
	defer req.Body.Close()
	return nil
}

// Get 讀取 private bucket 中的原始檔案。
func (s *SupabaseObjectStorage) Get(ctx context.Context, path string) ([]byte, error) {
	resp, err := s.request(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read object storage response: %w", err)
	}
	return content, nil
}

// Delete 僅用於資料庫交易失敗後清理已上傳的孤兒物件。
func (s *SupabaseObjectStorage) Delete(ctx context.Context, path string) error {
	resp, err := s.request(ctx, http.MethodDelete, path, nil, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

var _ app.ObjectStorage = (*SupabaseObjectStorage)(nil)
