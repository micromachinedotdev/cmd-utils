package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var targets = []struct {
	GOOS   string
	GOARCH string
	NPMPkg string
}{
	{"darwin", "arm64", "darwin-arm64"},
	{"darwin", "amd64", "darwin-x64"},
	{"linux", "arm64", "linux-arm64"},
	{"linux", "arm", "linux-arm"},
	{"linux", "amd64", "linux-x64"},
	{"windows", "arm64", "win32-arm64"},
	{"windows", "amd64", "win32-x64"},
}

func main() {
	version := os.Args[1]

	for _, t := range targets {
		fmt.Printf("Building %s/%s...\n", t.GOOS, t.GOARCH)

		binName := "micromachine"
		if t.GOOS == "windows" {
			binName = "micromachine.exe"
		}

		outDir := filepath.Join("npm", "@micromachine.dev", t.NPMPkg, "bin")
		err := os.MkdirAll(outDir, 0755)
		if err != nil {
			panic(err)
		}

		cmd := exec.Command("go", "build",
			"-ldflags", fmt.Sprintf("-s -w -X main.Version=%s", version),
			"-o", filepath.Join(outDir, binName),
			"./main.go",
		)
		cmd.Env = append(os.Environ(),
			"GOOS="+t.GOOS,
			"GOARCH="+t.GOARCH,
			"CGO_ENABLED=0",
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			panic(err)
		}
	}
}
