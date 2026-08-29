package utils

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreBase64Image(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	payload := []byte("fake png payload")
	url, err := StoreBase64Image(base64.StdEncoding.EncodeToString(payload))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, "/attachments/") || !strings.HasSuffix(url, ".png") {
		t.Fatalf("unexpected attachment URL: %q", url)
	}
	filename := strings.TrimPrefix(url, "/attachments/")
	stored, err := os.ReadFile(filepath.Join("storage", "attachments", filename))
	if err != nil {
		t.Fatal(err)
	}
	if string(stored) != string(payload) {
		t.Fatalf("stored payload = %q, want %q", stored, payload)
	}
}
