package utils

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type PackageJSON struct {
	Name                 string            `json:"name,omitempty"`
	Version              string            `json:"version,omitempty"`
	Private              bool              `json:"private,omitempty"`
	Type                 string            `json:"type,omitempty"` // "module" or "commonjs"
	Main                 string            `json:"main,omitempty"`
	Module               string            `json:"module,omitempty"`
	Types                string            `json:"types,omitempty"`
	Exports              json.RawMessage   `json:"exports,omitempty"` // complex, can be string or object
	Scripts              map[string]string `json:"scripts,omitempty"`
	Dependencies         map[string]string `json:"dependencies,omitempty"`
	DevDependencies      map[string]string `json:"devDependencies,omitempty"`
	PeerDependencies     map[string]string `json:"peerDependencies,omitempty"`
	OptionalDependencies map[string]string `json:"optionalDependencies,omitempty"`
	Engines              map[string]string `json:"engines,omitempty"`
	PackageManager       string            `json:"packageManager,omitempty"` // e.g., "pnpm@8.6.0"
	Workspaces           json.RawMessage   `json:"workspaces,omitempty"`     // can be []string or object
}

func (p *PackageJSON) HasDependency(name string) bool {
	if _, ok := p.Dependencies[name]; ok {
		return true
	}
	if _, ok := p.DevDependencies[name]; ok {
		return true
	}
	return false
}

func IsViteApp(rootDir string) (path *string, ok bool) {
	nextConfigPaths := []string{
		filepath.Join(rootDir, "vite.config.mjs"),
		filepath.Join(rootDir, "vite.config.js"),
		filepath.Join(rootDir, "vite.config.ts"),
	}

	for _, path := range nextConfigPaths {
		if _, err := os.Stat(path); err == nil {
			return &path, true
		}
	}

	return nil, false
}

type VitePluginChecks struct {
	HasDependency bool
	IsPlugin      bool
	Path          *string
}

func HasCloudflareVitePlugin(rootDir string) (checks *VitePluginChecks, ok bool) {
	path, ok := IsViteApp(rootDir)
	if !ok {
		return nil, false
	}

	data, err := os.ReadFile(filepath.Join(rootDir, "package.json"))
	if err != nil {
		return nil, false
	}

	var packageJSON PackageJSON
	err = json.Unmarshal(data, &packageJSON)
	if err != nil {
		return nil, false
	}

	checks = &VitePluginChecks{
		HasDependency: packageJSON.HasDependency("@cloudflare/vite-plugin"),
		Path:          path,
	}

	data, err = os.ReadFile(*path)
	if err != nil {
		return nil, false
	}

	plugins, err := DetectVitePluginsViaNode(rootDir)
	if err != nil {
		return nil, false
	}

	for _, plugin := range plugins {
		if strings.Contains(plugin, "vite-plugin-cloudflare") || strings.HasPrefix(plugin, "vite-plugin-cloudflare") {
			checks.IsPlugin = true
			return checks, true
		}
	}

	return nil, false
}

func DetectVitePluginsViaNode(projectDir string) ([]string, error) {
	script := `
import { resolveConfig } from 'vite';
const config = await resolveConfig({}, 'build');
console.log(JSON.stringify(config.plugins.map(p => p.name)));
`
	cmd := exec.Command("node", "--input-type=module", "-e", script)
	cmd.Dir = projectDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var names []string
	if err := json.Unmarshal(out, &names); err != nil {
		return nil, err
	}
	return names, nil
}

func IncludeCloudflareVitePlugin(rootDir, packageManager string) *string {
	checks, ok := HasCloudflareVitePlugin(rootDir)
	fmt.Println(checks, ok)
	if ok || checks == nil {
		return nil
	}

	if !checks.HasDependency {
		slog.Warn("We couldn't find the @cloudflare/vite-plugin-cloudflare dependency in your package.json. Adding it now...")
		cmd := exec.Command(packageManager, "install", "-D", "@cloudflare/vite-plugin-cloudflare")
		cmd.Dir = rootDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()

		if err != nil {
			slog.Error("Failed to add `@cloudflare/vite-plugin-cloudflare` vite plugin")
			os.Exit(1)
		}
	}

	if !checks.IsPlugin {
		slog.Warn("We couldn't find the @cloudflare/vite-plugin-cloudflare plugin in your vite.config.js. Adding it now...")
		return AddVitePlugin(rootDir, *checks.Path, packageManager)
	}
	return nil
}

func AddVitePlugin(rootDir, configPath, packageManager string) *string {
	script := fmt.Sprintf(`
		import { cloudflare } from '@cloudflare/vite-plugin-cloudflare';

		import userConfig from '%s';
		import { defineConfig, mergeConfig } from 'vite';

		export default mergeConfig(userConfig, defineConfig({
		plugins: [cloudflare()]
	}));`, configPath)

	path := filepath.Join(rootDir, "micromachine-vite.config.ts")
	err := os.WriteFile(path, []byte(script), 0644)
	if err != nil {
		slog.Error("Could not not add `@cloudflare/vite-plugin-cloudflare` to your vite.config.ts")
		os.Exit(1)
	}

	return &path
}
