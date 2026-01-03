package lib

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/evanw/esbuild/pkg/api"
)

type Bundle struct {
	RootDir        string
	ModulePath     string
	PackageManager string
	AssetPath      string
	BuildScript    string
	WrangleConfig  map[string]any
}

func (b *Bundle) Pack() {
	shouldBundle := IsOpenNext(b.WrangleConfig)
	if shouldBundle {
		result := api.Build(api.BuildOptions{
			EntryPoints: []string{b.RootDir + "/" + strings.TrimPrefix(b.ModulePath, "/")},
			Outdir:      b.GetModuleDir(),
			Bundle:      true,
			Write:       true,
			Splitting:   true,
			LogLevel:    api.LogLevelInfo,
			Format:      api.FormatESModule,
			Platform:    api.PlatformNode,
			TreeShaking: api.TreeShakingTrue,

			// Drop console/debugger
			Drop: api.DropConsole | api.DropDebugger,

			// Target modern runtime (Cloudflare Workers)
			Target: api.ESNext,

			External:          []string{"node:*", "cloudflare:*", "@cloudflare/*"},
			MinifyWhitespace:  true,
			MinifyIdentifiers: true,
			MinifySyntax:      true,
			Metafile:          true,
		})

		if len(result.Errors) > 0 {
			for _, err := range result.Errors {
				LogWithColor(Fail, fmt.Sprintf("✗ %v", err))
			}

			os.Exit(1)
		}
	}

	now := time.Now()
	LogWithColor(Default, "Copying assets...")
	err := copyDir(b.RootDir+"/"+strings.TrimPrefix(b.AssetPath, "/"), b.GetAssetDir())
	if err != nil {
		LogWithColor(Fail, fmt.Sprintf("✗ %v", err))
		os.Exit(1)
	}

	if !shouldBundle {
		err := copyDir(b.RootDir+"/"+strings.TrimPrefix(b.ModulePath, "/"), b.GetModuleDir())
		if err != nil {
			LogWithColor(Fail, fmt.Sprintf("✗ %v", err))
			os.Exit(1)
		}
	}
	elapsed := time.Since(now)
	LogWithColor(Success, fmt.Sprintf("✓ Assets copied in %s", elapsed))
}

func (b *Bundle) RunBuildCommand() {

	// Capture stdout and stderr separately
	cmd := exec.Command(b.PackageManager, "run", b.BuildScript)
	cmd.Dir = b.RootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if err != nil {
		LogWithColor(Fail, fmt.Sprintf("✗ %v", err))
		os.Exit(1)
	}
}

func (b *Bundle) GetOutputDir() string {
	return b.RootDir + "/.micromachine"
}

func (b *Bundle) GetModuleDir() string {
	return b.GetOutputDir() + "/worker"
}

func (b *Bundle) GetAssetDir() string {
	return b.GetOutputDir() + "/assets"
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, _ := d.Info()
		return os.WriteFile(target, data, info.Mode())
	})
}
