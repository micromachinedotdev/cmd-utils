package utils

import (
	"os"
	"path/filepath"
)

func IsOpenNext(wranglerConf map[string]any) bool {
	if wranglerConf["main"] == ".open-next/worker.js" {
		return true
	}

	if assets, ok := wranglerConf["assets"].(map[string]any); ok {
		if dir, ok := assets["directory"].(string); ok {
			if dir == ".open-next/assets" {
				return true
			}
		}
	}

	if _, err := os.Stat(".open-next"); err == nil {
		return true
	}

	return false
}

func IsNextJS(rootDir string) bool {
	nextConfigPaths := []string{
		filepath.Join(rootDir, "next.config.mjs"),
		filepath.Join(rootDir, "next.config.js"),
		filepath.Join(rootDir, "next.config.ts"),
	}

	for _, path := range nextConfigPaths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	if stat, err := os.Stat(".next"); !os.IsNotExist(err) && !stat.IsDir() {
		return true
	}

	return false
}
