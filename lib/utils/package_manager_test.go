package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectPackageManager(t *testing.T) {

	tests := []struct {
		name             string
		expected         string
		lockfile         string
		usingPackageJson bool
	}{
		{"Detect Bun package manager", "bun", "bun.lock", false},
		{"Detect Bun package manager using package.json", "bun", "bun.lock", true},
		{"Detect Yarn package manager", "yarn", "yarn.lock", false},
		{"Detect NPM package manager", "npm", "package-lock.json", false},
		{"Detect Pnpm package manager", "pnpm", "pnpm-lock.yaml", false},
		{"Detect Pnpm package manager using package.json", "pnpm", "pnpm-lock.yaml", true},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			t.Cleanup(func() {
				err := os.RemoveAll(dir)
				if err != nil {
					return
				}
			})

			if tt.usingPackageJson {
				err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(fmt.Sprintf(`{"name": "test","packageManager": "%s"}`, tt.expected)), 0644)
				if err != nil {
					t.Error(err)
				}
			} else {
				err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name": "test"}`), 0644)
				if err != nil {
					t.Error(err)
				}
			}

			ext := filepath.Ext(tt.lockfile)
			switch ext {
			case ".lock":
				err := os.WriteFile(filepath.Join(dir, tt.lockfile), []byte(`{"lockfileVersion": 1,"configVersion": 1}`), 0644)
				if err != nil {
					t.Error(err)
				}
			case ".yaml":
				err := os.WriteFile(filepath.Join(dir, tt.lockfile), []byte(`lockfileVersion: '9.0'`), 0644)
				if err != nil {
					t.Error(err)
				}
			case ".json":
				err := os.WriteFile(filepath.Join(dir, tt.lockfile), []byte(`{}`), 0644)
				if err != nil {
					t.Error(err)
				}
			}

			got, err := DetectPackageManager(&dir)
			if err != nil {
				t.Errorf("expected %s, got error: %v", tt.expected, err)
				return
			}

			if got == nil {
				t.Errorf("expected %s, got nil", tt.expected)
				return
			}

			if *got != tt.expected {
				t.Errorf("DetectPackageManager(%s) = %s, want %s", dir, *got, tt.expected)
			}
		})
	}
}

func TestMissingPackageManager(t *testing.T) {
	dir := t.TempDir()
	_, err := DetectPackageManager(&dir)
	if err == nil {
		t.Error("Expected error when no package manager found")
	}
}
