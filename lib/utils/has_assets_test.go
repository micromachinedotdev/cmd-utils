package utils

import "testing"

func TestHasAssets(t *testing.T) {
	conf := &WranglerConfig{
		Main: ".open-next/worker.js",
		Name: "testing",
		Assets: &AssetsConfig{
			Directory: ".open-next/assets",
		},
	}

	result := HasAssets(conf)
	if result != true {
		t.Errorf("Expected `HasAssets` to return true, got %v", result)
	}
}

func TestWranglerConfigWithoutAssets(t *testing.T) {
	conf := &WranglerConfig{
		Main: ".open-next/worker.js",
		Name: "testing",
	}
	result := HasAssets(conf)
	if result != false {
		t.Errorf("Expected `HasAssets` to return false, got %v", result)
	}
}
