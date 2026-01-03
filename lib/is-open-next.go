package lib

import "os"

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
