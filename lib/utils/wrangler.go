package utils

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/tidwall/jsonc"
)

func DetectWranglerFile(root *string) (map[string]any, error) {
	rootDir := ""
	if root != nil {
		rootDir = *root
	}

	paths := []string{rootDir + "/wrangler.toml", rootDir + "/wrangler.json", rootDir + "/wrangler.jsonc"}

	content := ""
	usedPath := ""

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}

		usedPath = path
		content = string(data)
	}

	if content == "" || usedPath == "" {
		return nil, errors.New("no wrangler configuration file found")
	}

	ext := filepath.Ext(usedPath)
	if ext == "" {
		return nil, errors.New("invalid wrangler configuration file")
	}

	config := map[string]any{}
	switch ext {
	case ".json", ".jsonc":
		err := json.Unmarshal(jsonc.ToJSON([]byte(content)), &config)
		if err != nil {
			return nil, err
		}
	case ".toml":
		err := toml.Unmarshal([]byte(content), &config)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("invalid wrangler configuration file")
	}

	return config, nil
}
