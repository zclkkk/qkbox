import { spawnSync } from "node:child_process";

const paths = ["apps", "cmd", "core", "internal", "platform", "shared", "test"];
const result = spawnSync("gofmt", ["-l", ...paths], {
  encoding: "utf8",
  shell: false
});

if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}

if (result.status !== 0) {
  if (result.stderr) {
    process.stderr.write(result.stderr);
  }
  process.exit(result.status ?? 1);
}

const unformatted = result.stdout.trim();
if (unformatted !== "") {
  console.error("Go files need gofmt:");
  console.error(unformatted);
  console.error("Run: npm run go:fmt");
  process.exit(1);
}
