import { existsSync } from "node:fs";
import { dirname, join } from "node:path";
import { platform } from "node:os";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const rootDir = dirname(dirname(fileURLToPath(import.meta.url)));
const host = platform();

function run(args) {
  const command = host === "win32" ? "cmd.exe" : "npm";
  const commandArgs = host === "win32" ? ["/d", "/s", "/c", "npm", ...args] : args;
  console.log(`> npm ${args.join(" ")}`);
  const result = spawnSync(command, commandArgs, {
    cwd: rootDir,
    stdio: "inherit",
    shell: false
  });
  if (result.error) {
    console.error(result.error.message);
    process.exit(1);
  }
  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
}

function requireFile(path) {
  if (!existsSync(path)) {
    console.error(`Missing required release artifact: ${path}`);
    process.exit(1);
  }
}

run(["run", "check"]);
run(["run", "build"]);

const suffix = host === "win32" ? ".exe" : "";
for (const binary of ["qkbox", "qkbox-window", "qkbox-provider"]) {
  requireFile(join(rootDir, "bin", `${binary}${suffix}`));
}

switch (host) {
  case "win32":
    run(["run", "package:windows"]);
    break;
  case "linux":
    run(["run", "package:linux"]);
    break;
  case "darwin":
    run(["run", "package:macos"]);
    break;
  default:
    console.error(`Unsupported release verification host: ${host}`);
    process.exit(1);
}
