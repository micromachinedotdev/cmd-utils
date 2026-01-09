package lib

import "testing"

func TestWranglerConfigForOpenNext(t *testing.T) {
	conf := map[string]any{
		"main": ".open-next/worker.js",
		"name": "testing",
		"assets": map[string]any{
			"directory": ".open-next/assets",
		},
	}
	result := IsOpenNext(conf)
	if result != true {
		t.Errorf("Expected `IsOpenNext` to return true, got %v", result)
	}
}

func TestWranglerConfigForNonOpenNext(t *testing.T) {
	conf := map[string]any{
		"main": "dist/worker.js",
		"name": "testing",
		"assets": map[string]any{
			"directory": "dis/assets",
		},
	}
	result := IsOpenNext(conf)
	if result != false {
		t.Errorf("Expected `IsOpenNext` to return false, got %v", result)
	}
}
