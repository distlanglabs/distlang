package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFile(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "index.js")
	content := "export default { fetch() { return new Response(\"ok\") } }\n"

	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got, err := ParseFile(filePath)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}

	if got != content {
		t.Fatalf("unexpected content\nwant: %q\n got: %q", content, got)
	}
}

func TestParseFileMissing(t *testing.T) {
	_, err := ParseFile("does-not-exist.js")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
