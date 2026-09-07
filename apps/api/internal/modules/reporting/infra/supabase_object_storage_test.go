package infra

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSupabaseObjectStorageRoundTrip(t *testing.T) {
	var (
		methods       []string
		requestBody   []byte
		authorization string
		apiKey        string
		upsert        string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.URL.Path != "/storage/v1/object/ltc-exports/exports/job-1/report.xlsx" {
			http.Error(w, "unexpected path", http.StatusBadRequest)
			return
		}
		authorization = r.Header.Get("Authorization")
		apiKey = r.Header.Get("apikey")
		if r.Method == http.MethodPost {
			requestBody, _ = io.ReadAll(r.Body)
			upsert = r.Header.Get("x-upsert")
		}
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte("stored-xlsx"))
		}
	}))
	defer server.Close()

	storage := NewSupabaseObjectStorage(server.URL, "service-key", "ltc-exports", server.Client())
	ctx := context.Background()
	if err := storage.Put(ctx, "exports/job-1/report.xlsx", "application/test", []byte("uploaded-xlsx")); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	got, err := storage.Get(ctx, "exports/job-1/report.xlsx")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if string(got) != "stored-xlsx" {
		t.Fatalf("Get() = %q, want stored-xlsx", got)
	}
	if err := storage.Delete(ctx, "exports/job-1/report.xlsx"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if string(requestBody) != "uploaded-xlsx" {
		t.Errorf("Put() body = %q, want uploaded-xlsx", requestBody)
	}
	if authorization != "Bearer service-key" || apiKey != "service-key" {
		t.Errorf("service headers = (%q, %q), want service-key", authorization, apiKey)
	}
	if upsert != "false" {
		t.Errorf("x-upsert = %q, want false", upsert)
	}
	wantMethods := []string{http.MethodPost, http.MethodGet, http.MethodDelete}
	if len(methods) != len(wantMethods) {
		t.Fatalf("methods = %v, want %v", methods, wantMethods)
	}
	for i := range wantMethods {
		if methods[i] != wantMethods[i] {
			t.Errorf("methods[%d] = %q, want %q", i, methods[i], wantMethods[i])
		}
	}
}

func TestSupabaseObjectStorageRejectsInvalidPath(t *testing.T) {
	storage := NewSupabaseObjectStorage("https://example.supabase.co", "service-key", "ltc-exports", nil)
	for _, path := range []string{"", "exports/../secret.xlsx", "exports//secret.xlsx"} {
		if err := storage.Put(context.Background(), path, "application/test", nil); err == nil {
			t.Errorf("Put(%q) error = nil, want invalid path error", path)
		}
	}
}

func TestSupabaseObjectStorageReturnsHTTPErrorWithoutBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "contains internal details", http.StatusForbidden)
	}))
	defer server.Close()

	storage := NewSupabaseObjectStorage(server.URL, "service-key", "ltc-exports", server.Client())
	_, err := storage.Get(context.Background(), "exports/job-1/report.xlsx")
	if err == nil {
		t.Fatal("Get() error = nil, want HTTP error")
	}
	if got := err.Error(); got != "object storage returned HTTP 403" {
		t.Fatalf("Get() error = %q, want stable HTTP error", got)
	}
}
