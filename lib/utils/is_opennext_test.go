package utils

import "testing"

func TestWranglerConfigForOpenNext(t *testing.T) {
	conf := &WranglerConfig{
		Main: ".open-next/worker.js",
		Name: "testing",
		Assets: &AssetsConfig{
			Directory: ".open-next/assets",
		},
	}
	result := IsOpenNext(conf)
	if result != true {
		t.Errorf("Expected `IsOpenNext` to return true, got %v", result)
	}
}

func TestWranglerConfigForNonOpenNext(t *testing.T) {
	conf := &WranglerConfig{
		Main: "dist/worker.js",
		Name: "testing",
		Assets: &AssetsConfig{
			Directory: "dis/assets",
		},
	}
	result := IsOpenNext(conf)
	if result != false {
		t.Errorf("Expected `IsOpenNext` to return false, got %v", result)
	}
}
