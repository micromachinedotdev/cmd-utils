#!/usr/bin/env node

const { execFileSync } = require("child_process");
const path = require("path");
const fs = require("fs");

function getBinaryPath() {
  const platform = process.platform;
  const arch = process.arch;

  // Map Node.js platform/arch to your package names
  const platformMap = {
    darwin: "darwin",
    linux: "linux",
    win32: "windows",
  };

  const archMap = {
    arm64: "arm64",
    x64: "x64",
  };

  const os = platformMap[platform];
  const cpu = archMap[arch];

  if (!os || !cpu) {
    throw new Error(`Unsupported platform: ${platform}-${arch}`);
  }

  const binaryName = platform === "win32" ? "micromachine.exe" : "micromachine";
  const pkgName = `@micromachine.dev/cli-${os}-${cpu}`;

  try {
    // Try to find the binary in the platform-specific package
    const pkgPath = require.resolve(`${pkgName}/package.json`);
    const binPath = path.join(path.dirname(pkgPath), "bin", binaryName);

    if (fs.existsSync(binPath)) {
      return binPath;
    }
  } catch (e) {
    // Package not found
  }

  throw new Error(
      `Could not find binary for ${platform}-${arch}. ` +
      `Please ensure ${pkgName} is installed.`
  );
}

const binary = getBinaryPath();
const args = process.argv.slice(2);

try {
  execFileSync(binary, args, { stdio: "inherit" });
} catch (error) {
  if (error.status !== undefined) {
    process.exit(error.status);
  }
  throw error;
}