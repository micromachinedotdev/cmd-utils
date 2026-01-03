# Micromachine CLI (cmd-utils)

Micromachine CLI provides helper commands for building and packaging web apps to run on edge/runtime targets.

Currently, the main command is `micromachine build`, which detects your package manager, reads your Wrangler configuration, builds your app, and bundles artifacts.

## Install

- Using Go (local):

```bash
go run ./cmd-utils
```

- Build a binary:

```bash
cd cmd-utils
go build -o micromachine
./micromachine --help
```

## Usage

Show help:

```bash
micromachine --help
```

Build your project (from the project root by default):

```bash
micromachine build
```

Flags:
- `-r, --rootdir` Path to the app root (default: `.`)
- `-b, --build-script` Package manager script to run before bundling (default: `build`)

Example:

```bash
micromachine build -r ./apps/hello-world -b build
```

## Wrangler configuration

The CLI looks for a Wrangler config in the root directory in the following order:
`wrangler.toml`, `wrangler.json`, `wrangler.jsonc`.

Expected fields used by the builder:
- `main` (string): Module entrypoint (e.g., `./.micromachine/worker/handler.js`).
- `assets.directory` (string, optional): Path to static assets directory.

If no configuration is found or `main` is missing, the command will fail with an error.

## Development

Run locally while developing:

```bash
go run ./cmd-utils
```

Run the build command directly with flags:

```bash
go run ./cmd-utils build -r ./example
```

## License

Copyright © 2026 Micromachine
