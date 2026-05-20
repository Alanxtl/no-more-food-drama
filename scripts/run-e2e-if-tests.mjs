import { spawnSync } from "node:child_process";
import { existsSync, readdirSync } from "node:fs";
import { join } from "node:path";

const testDir = "e2e";
const testFilePattern = /\.(spec|test)\.(js|jsx|ts|tsx|mjs|cjs)$/;

function hasE2eTests(directory) {
  if (!existsSync(directory)) {
    return false;
  }

  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    const fullPath = join(directory, entry.name);

    if (entry.isDirectory() && hasE2eTests(fullPath)) {
      return true;
    }

    if (entry.isFile() && testFilePattern.test(entry.name)) {
      return true;
    }
  }

  return false;
}

if (!hasE2eTests(testDir)) {
  console.log("No e2e tests found; skipping Playwright.");
  process.exit(0);
}

const result = spawnSync("npx", ["playwright", "test"], {
  stdio: "inherit",
  shell: true
});

process.exit(result.status ?? 1);
