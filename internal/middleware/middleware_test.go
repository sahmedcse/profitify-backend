package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// newCapturingLogger returns a logger writing JSON into buf.
func newCapturingLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func TestRequestLogger_LogsRequestFields(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		status     int
		body       string
		wantStatus int
		wantBytes  int
	}{
		{name: "ok get", method: http.MethodGet, path: "/health", status: http.StatusOK, body: "hello", wantStatus: 200, wantBytes: 5},
		{name: "not found", method: http.MethodGet, path: "/missing", status: http.StatusNotFound, body: "", wantStatus: 404, wantBytes: 0},
		{name: "post created", method: http.MethodPost, path: "/things", status: http.StatusCreated, body: "abc", wantStatus: 201, wantBytes: 3},
		{name: "server error", method: http.MethodGet, path: "/boom", status: http.StatusInternalServerError, body: "x", wantStatus: 500, wantBytes: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			mw := RequestLogger(newCapturingLogger(&buf))

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				if tt.body != "" {
					_, _ = w.Write([]byte(tt.body))
				}
			})

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			mw(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("response status = %d, want %d", rec.Code, tt.wantStatus)
			}

			var logged map[string]any
			if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &logged); err != nil {
				t.Fatalf("log line is not valid JSON: %v (%q)", err, buf.String())
			}

			if logged["msg"] != "request" {
				t.Errorf("msg = %v, want %q", logged["msg"], "request")
			}
			if logged["method"] != tt.method {
				t.Errorf("method = %v, want %v", logged["method"], tt.method)
			}
			if logged["path"] != tt.path {
				t.Errorf("path = %v, want %v", logged["path"], tt.path)
			}
			if got := int(logged["status"].(float64)); got != tt.wantStatus {
				t.Errorf("logged status = %d, want %d", got, tt.wantStatus)
			}
			if got := int(logged["bytes"].(float64)); got != tt.wantBytes {
				t.Errorf("logged bytes = %d, want %d", got, tt.wantBytes)
			}
			if _, ok := logged["duration"]; !ok {
				t.Error("duration field missing from log line")
			}
			if _, ok := logged["request_id"]; !ok {
				t.Error("request_id field missing from log line")
			}
		})
	}
}

func TestRequestLogger_IncludesRequestIDFromContext(t *testing.T) {
	var buf bytes.Buffer
	mw := RequestLogger(newCapturingLogger(&buf))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// chi's RequestID middleware populates the context value RequestLogger reads.
	handler := chimw.RequestID(mw(next))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var logged map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &logged); err != nil {
		t.Fatalf("log line is not valid JSON: %v", err)
	}

	reqID, _ := logged["request_id"].(string)
	if reqID == "" {
		t.Error("request_id should be populated when chi RequestID middleware runs")
	}
}

func TestRequestLogger_LogsWhenHandlerWritesNothing(t *testing.T) {
	var buf bytes.Buffer
	mw := RequestLogger(newCapturingLogger(&buf))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	req := httptest.NewRequest(http.MethodGet, "/quiet", nil)
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)

	if buf.Len() == 0 {
		t.Fatal("expected a log line even when the handler writes nothing")
	}
}
