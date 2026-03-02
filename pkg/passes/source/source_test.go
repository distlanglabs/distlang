package source

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFile(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "index.js")
	content := "export default { fetch() { return new Response(\"ok\") } }\n"

	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got, err := ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	if got != content {
		t.Fatalf("unexpected content\nwant: %q\n got: %q", content, got)
	}
}

func TestReadFileMissing(t *testing.T) {
	_, err := ReadFile("does-not-exist.js")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
