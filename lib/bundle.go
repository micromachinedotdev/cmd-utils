package lib

import (
	"encoding/json"
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
	Environment    string
	WrangleConfig  map[string]any
}

func (b *Bundle) Pack() {
	shouldBundle := IsOpenNext(b.WrangleConfig)
	if shouldBundle {

		var warnedPackages = make(map[string][]string)

		nodeCompatPlugin := createNodeCompatPlugin()

		var cfPaths = make(map[string]struct{})

		cloudflarePlugin := api.Plugin{
			Name: "cloudflare-internal",
			Setup: func(build api.PluginBuild) {
				build.OnResolve(api.OnResolveOptions{Filter: "^cloudflare:"},
					func(args api.OnResolveArgs) (api.OnResolveResult, error) {
						cfPaths[args.Path] = struct{}{}
						return api.OnResolveResult{External: true}, nil
					})
			},
		}

		result := api.Build(api.BuildOptions{
			Plugins:     []api.Plugin{nodeCompatPlugin, cloudflarePlugin},
			EntryPoints: []string{b.RootDir + "/" + strings.TrimPrefix(b.ModulePath, "/")},
			Outdir:      b.GetModuleDir(),
			Bundle:      true,
			Write:       true,
			Splitting:   false,
			LogLevel:    api.LogLevelInfo,
			Format:      api.FormatDefault,
			Platform:    api.PlatformNeutral,
			TreeShaking: api.TreeShakingTrue,
			Loader:      map[string]api.Loader{".js": api.LoaderJSX, ".mjs": api.LoaderJSX, ".cjs": api.LoaderJSX},

			// Drop console/debugger
			//Drop: api.DropConsole | api.DropDebugger,

			// Target modern runtime (Cloudflare Workers)
			Target: api.ES2024,

			External:          []string{"__STATIC_CONTENT_MANIFEST"},
			MinifyWhitespace:  true,
			MinifyIdentifiers: true,
			MinifySyntax:      true,
			KeepNames:         true,
			Metafile:          true,
			Conditions:        []string{"workerd", "worker", "browser"},
			Define: map[string]string{
				"process.env.NODE_ENV":            toJSString(b.Environment),
				"global.process.env.NODE_ENV":     toJSString(b.Environment),
				"globalThis.process.env.NODE_ENV": toJSString(b.Environment),
			},
		})

		if len(result.Errors) > 0 {
			for _, err := range result.Errors {
				LogWithColor(Fail, fmt.Sprintf("✗ %v", err))
			}

			os.Exit(1)
		}

		// Later, after build:
		for path, importers := range warnedPackages {
			LogWithColor(Fail, fmt.Sprintf("Warning: Node builtin %q used (from %v)\n", path, importers))
		}

		// After build (check format if needed):

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

func toJSString(val string) string {
	if val == "" {
		return `""` // or "undefined" depending on your needs
	}
	// JSON marshal handles escaping
	b, _ := json.Marshal(val)
	return string(b)
}

var nodeBuiltins = map[string]bool{
	"async_hooks": true, "buffer": true, "crypto": true, "events": true,
	"fs": true, "http": true, "https": true, "os": true, "path": true,
	"stream": true, "tty": true, "url": true, "util": true, "vm": true,
	"zlib": true, "net": true, "child_process": true, "worker_threads": true,
	"assert": true, "constants": true, "dns": true, "domain": true,
	"module": true, "process": true, "punycode": true, "querystring": true,
	"readline": true, "repl": true, "string_decoder": true, "sys": true,
	"timers": true, "v8": true, "wasi": true,
}

func createNodeCompatPlugin() api.Plugin {
	seen := make(map[string]struct{})
	warnedPackages := make(map[string][]string)

	return api.Plugin{
		Name: "nodejs_compat",
		Setup: func(build api.PluginBuild) {
			// Clear state on each build
			build.OnStart(func() (api.OnStartResult, error) {
				seen = make(map[string]struct{})
				warnedPackages = make(map[string][]string)
				return api.OnStartResult{}, nil
			})

			// Handle node: prefixed imports
			build.OnResolve(api.OnResolveOptions{Filter: `^node:`},
				func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					// Composite key for deduplication
					specifier := fmt.Sprintf("%s:%d:%s:%s", args.Path, args.Kind, args.ResolveDir, args.Importer)
					if _, ok := seen[specifier]; ok {
						return api.OnResolveResult{}, nil
					}
					seen[specifier] = struct{}{}

					// Try to resolve as a normal package (maybe there's a polyfill)
					result := build.Resolve(args.Path, api.ResolveOptions{
						Kind:       args.Kind,
						ResolveDir: args.ResolveDir,
						Importer:   args.Importer,
					})

					if len(result.Errors) > 0 {
						// Not found locally, mark as external (runtime will provide it)
						warnedPackages[args.Path] = append(warnedPackages[args.Path], args.Importer)
						return api.OnResolveResult{External: true}, nil
					}

					// Found locally, use that
					return api.OnResolveResult{Path: result.Path}, nil
				})

			// Handle bare Node.js built-in imports (fs, path, etc.)
			build.OnResolve(api.OnResolveOptions{Filter: `^[a-z_]+$`},
				func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					if !nodeBuiltins[args.Path] {
						return api.OnResolveResult{}, nil
					}

					specifier := fmt.Sprintf("%s:%d:%s:%s", args.Path, args.Kind, args.ResolveDir, args.Importer)
					if _, ok := seen[specifier]; ok {
						return api.OnResolveResult{}, nil
					}
					seen[specifier] = struct{}{}

					// Try to resolve (maybe there's a polyfill installed)
					result := build.Resolve(args.Path, api.ResolveOptions{
						Kind:       args.Kind,
						ResolveDir: args.ResolveDir,
						Importer:   args.Importer,
					})

					if len(result.Errors) > 0 {
						warnedPackages[args.Path] = append(warnedPackages[args.Path], args.Importer)
						return api.OnResolveResult{External: true}, nil
					}

					return api.OnResolveResult{Path: result.Path}, nil
				})

			// Log warnings at the end
			//build.OnEnd(func(result *api.BuildResult) (api.OnEndResult, error) {
			//	for pkg, importers := range warnedPackages {
			//		//fmt.Printf("⚠️  Package %q is a Node.js built-in. Imported from:\n", pkg)
			//		for _, imp := range importers {
			//			fmt.Printf("   - %s\n", imp)
			//		}
			//	}
			//	return api.OnEndResult{}, nil
			//})
		},
	}
}
