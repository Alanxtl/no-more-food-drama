import { execFileSync, spawn } from "node:child_process";
import http from "node:http";
import { existsSync, mkdirSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const projectRoot = path.join(__dirname, "..");
const nextCLI = path.join(__dirname, "..", "node_modules", "next", "dist", "bin", "next");

const pathKey = process.platform === "win32" ? "Path" : "PATH";
const envPath = process.env[pathKey] ?? process.env.PATH ?? "";
const machinePath = windowsEnvironmentPath("Machine");
const userPath = windowsEnvironmentPath("User");
const fallbackPaths = process.platform === "win32" ? ["C:\\Program Files\\Go\\bin"] : [];
process.env[pathKey] = [envPath, machinePath, userPath, ...fallbackPaths].filter(Boolean).join(path.delimiter);
process.env.USE_MOCK_PROVIDERS = "true";
process.env.NEXT_PUBLIC_APP_URL = "http://127.0.0.1:3000";
process.env.E2E_API_URL = "http://127.0.0.1:3002";
const tmpDir = path.join(projectRoot, ".tmp");
if (!existsSync(tmpDir)) {
  mkdirSync(tmpDir);
}
const apiBinary = path.join(tmpDir, process.platform === "win32" ? "e2e-api-server.exe" : "e2e-api-server");
execFileSync("go", ["build", "-o", apiBinary, "./scripts/e2e-api-server.go"], {
  cwd: projectRoot,
  stdio: "inherit",
  env: process.env,
});

const children = [
  spawn(apiBinary, [], { cwd: projectRoot, stdio: "inherit", env: process.env }),
  spawn(process.execPath, [nextCLI, "dev", "--hostname", "127.0.0.1", "--port", "3000"], {
    cwd: projectRoot,
    stdio: "inherit",
    env: process.env,
  }),
];
const readinessServer = http.createServer(async (request, response) => {
  if (request.url !== "/ready") {
    response.writeHead(404);
    response.end();
    return;
  }

  const apiReady = await canFetch("http://127.0.0.1:3002/api/rooms");
  const nextReady = await canFetch("http://127.0.0.1:3000/");
  response.writeHead(apiReady && nextReady ? 200 : 503, { "Content-Type": "text/plain" });
  response.end(apiReady && nextReady ? "ready" : "not ready");
});
readinessServer.listen(3003, "127.0.0.1");

for (const child of children) {
  child.on("exit", (code) => {
    if (code && code !== 0) {
      shutdown();
      process.exit(code);
    }
  });
}

function shutdown() {
  readinessServer.close();
  for (const child of children) {
    if (!child.killed) {
      child.kill();
    }
  }
}

function canFetch(url) {
  return new Promise((resolve) => {
    const request = http.get(url, (response) => {
      response.resume();
      resolve(response.statusCode === 200 || response.statusCode === 404 || response.statusCode === 405);
    });
    request.on("error", () => resolve(false));
    request.setTimeout(1000, () => {
      request.destroy();
      resolve(false);
    });
  });
}

process.on("SIGINT", () => {
  shutdown();
  process.exit(130);
});

process.on("SIGTERM", () => {
  shutdown();
  process.exit(143);
});

function windowsEnvironmentPath(scope) {
  if (process.platform !== "win32") {
    return "";
  }

  try {
    return execFileSync(
      "powershell.exe",
      ["-NoProfile", "-Command", `[Environment]::GetEnvironmentVariable('Path','${scope}')`],
      { encoding: "utf8" },
    ).trim();
  } catch {
    return "";
  }
}
