package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func DetectPackageManager(root *string) (*string, error) {
	rootDir := ""
	if root != nil {
		rootDir = *root
	}

	data, err := os.ReadFile(rootDir + "/package.json")

	if err != nil {
		return nil, err
	}

	packageJson := map[string]any{}

	err = json.Unmarshal(data, &packageJson)
	if err != nil {
		return nil, err
	}

	if pm, ok := packageJson["packageManager"].(string); ok {
		before, _, found := strings.Cut(pm, "@")
		if found {
			LogWithColor(Default, fmt.Sprintf("Detected \033[1m`%s`\033[0m package manager", before))
			return &before, nil
		}
	}

	if _, err = os.Stat(rootDir + "/bun.lock"); err == nil {
		pm := "bun"
		LogWithColor(Default, fmt.Sprintf("Detected \033[1m`%s`\033[0m package manager", pm))
		return &pm, nil
	}

	if _, err = os.Stat(rootDir + "/pnpm-lock.yaml"); err == nil {
		pm := "pnpm"
		LogWithColor(Default, fmt.Sprintf("Detected \033[1m`%s`\033[0m package manager", pm))
		return &pm, nil
	}

	if _, err = os.Stat(rootDir + "/yarn.lock"); err == nil {
		pm := "yarn"
		LogWithColor(Default, fmt.Sprintf("Detected \033[1m`%s`\033[0m package manager", pm))
		return &pm, nil
	}

	npm := "npm"
	return &npm, nil
}
