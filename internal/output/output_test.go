package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestPrintJSON(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]string{"id": "pod123", "status": "running"}
	if err := Fprint(&buf, FormatJSON, data); err != nil {
		t.Fatalf("Fprint JSON error: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	if result["id"] != "pod123" {
		t.Errorf("got id=%q, want %q", result["id"], "pod123")
	}
}

func TestPrintYAML(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]string{"id": "pod123"}
	if err := Fprint(&buf, FormatYAML, data); err != nil {
		t.Fatalf("Fprint YAML error: %v", err)
	}

	if !strings.Contains(buf.String(), "id: pod123") {
		t.Errorf("YAML output should contain 'id: pod123', got: %s", buf.String())
	}
}

func TestPrintTable(t *testing.T) {
	var buf bytes.Buffer
	type item struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	data := []item{
		{ID: "pod1", Status: "running"},
		{ID: "pod2", Status: "stopped"},
	}
	if err := Fprint(&buf, FormatTable, data); err != nil {
		t.Fatalf("Fprint Table error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "ID") {
		t.Errorf("Table should contain header 'ID', got: %s", out)
	}
	if !strings.Contains(out, "pod1") {
		t.Errorf("Table should contain 'pod1', got: %s", out)
	}
}

func TestPrintErrorFormat(t *testing.T) {
	var buf bytes.Buffer
	err := Fprint(&buf, FormatJSON, ErrorResponse{
		Error: Error{Code: "test_error", Message: "something failed"},
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	var result ErrorResponse
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if result.Error.Code != "test_error" {
		t.Errorf("got code=%q, want %q", result.Error.Code, "test_error")
	}
}
