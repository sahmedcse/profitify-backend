package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestRun_LogsStartAndFinish(t *testing.T) {
	var buf bytes.Buffer
	run(slog.New(slog.NewJSONHandler(&buf, nil)))

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d log lines, want 2: %q", len(lines), buf.String())
	}

	want := []string{"cron runner started", "cron runner finished"}
	for i, line := range lines {
		var logged map[string]any
		if err := json.Unmarshal([]byte(line), &logged); err != nil {
			t.Fatalf("line %d is not valid JSON: %v", i, err)
		}
		if logged["msg"] != want[i] {
			t.Errorf("line %d msg = %v, want %q", i, logged["msg"], want[i])
		}
	}
}
