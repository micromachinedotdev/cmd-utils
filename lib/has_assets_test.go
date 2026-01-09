package lib

import "testing"

func TestHasAssets(t *testing.T) {
	conf := map[string]any{
		"main": ".open-next/worker.js",
		"name": "testing",
		"assets": map[string]any{
			"directory": ".open-next/assets",
		},
	}

	result := HasAssets(conf)
	if result != true {
		t.Errorf("Expected `HasAssets` to return true, got %v", result)
	}
}

func TestWranglerConfigWithoutAssets(t *testing.T) {
	conf := map[string]any{
		"main": "dist/worker.js",
		"name": "testing",
	}
	result := HasAssets(conf)
	if result != false {
		t.Errorf("Expected `HasAssets` to return false, got %v", result)
	}
}
