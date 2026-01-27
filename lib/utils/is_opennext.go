package utils

import (
	"os"
	"path/filepath"
)

func IsOpenNext(wranglerConf *WranglerConfig) bool {
	if wranglerConf != nil && wranglerConf.Main == ".open-next/worker.js" {
		return true
	}

	if wranglerConf != nil && wranglerConf.Assets != nil && wranglerConf.Assets.Directory == ".open-next/assets" {
		return true
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
