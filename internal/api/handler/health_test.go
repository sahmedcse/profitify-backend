package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status field = %q, want %q", body["status"], "ok")
	}
}

// failingWriter forces the json encoder to error so the error branch is exercised.
type failingWriter struct {
	header http.Header
	code   int
}

func (f *failingWriter) Header() http.Header {
	if f.header == nil {
		f.header = make(http.Header)
	}
	return f.header
}
func (f *failingWriter) Write([]byte) (int, error) { return 0, errWriteFailed }
func (f *failingWriter) WriteHeader(code int)      { f.code = code }

var errWriteFailed = &writeError{}

type writeError struct{}

func (e *writeError) Error() string { return "write failed" }

func TestHealth_EncodeError(t *testing.T) {
	fw := &failingWriter{}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	// Must not panic when the response cannot be written.
	Health(fw, req)

	if fw.code != http.StatusOK {
		t.Errorf("status = %d, want %d", fw.code, http.StatusOK)
	}
}
