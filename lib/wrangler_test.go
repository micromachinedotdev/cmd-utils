package lib

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWranglerDetection(t *testing.T) {
	tests := []struct {
		name string
		ext  string
	}{
		{"Detect wrangler.jsonc", ".jsonc"},
		{"Detect wrangler.json", ".json"},
		{"Detect wrangler.toml", ".toml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := os.TempDir()

			t.Cleanup(func() {
				err := os.RemoveAll(dir)
				if err != nil {
					return
				}
			})

			switch tt.ext {
			case ".toml":
				err := os.WriteFile(filepath.Join(dir, "wrangler.toml"), []byte(`name = "test"`), 0644)
				if err != nil {
					t.Error(err)
				}
			case ".jsonc", ".json":
				err := os.WriteFile(filepath.Join(dir, "wrangler.json"), []byte(`{"name": "test"}`), 0644)
				if err != nil {
					t.Error(err)
				}
			}
			got, err := DetectWranglerFile(&dir)

			if err != nil {
				t.Errorf("expected wrangler map, got error: %v", err)
				return
			}

			if got == nil {
				t.Error("expected `map[string]any`, got nil")
				return
			}

			name, ok := got["name"].(string)

			if !ok || name != "test" {
				t.Errorf("Expected wrangler config `name` to be %s, got %s", "test", name)
			}
		})
	}
}

func TestMissingWranglerFile(t *testing.T) {
	dir := os.TempDir()
	_, err := DetectWranglerFile(&dir)
	if err == nil {
		t.Error("Expected error when no wrangler file found")
	}
}
