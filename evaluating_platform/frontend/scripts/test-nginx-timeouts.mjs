import { readFileSync } from "node:fs";
import { resolve } from "node:path";

const nginxConfig = readFileSync(resolve("nginx.conf"), "utf8");

function locationBlock(pattern) {
  const start = nginxConfig.indexOf(pattern);
  if (start === -1) {
    throw new Error(`missing ${pattern} location`);
  }

  const open = nginxConfig.indexOf("{", start);
  if (open === -1) {
    throw new Error(`missing opening brace for ${pattern}`);
  }

  let depth = 0;
  for (let i = open; i < nginxConfig.length; i += 1) {
    const char = nginxConfig[i];
    if (char === "{") {
      depth += 1;
    } else if (char === "}") {
      depth -= 1;
      if (depth === 0) {
        return nginxConfig.slice(open + 1, i);
      }
    }
  }

  throw new Error(`missing closing brace for ${pattern}`);
}

function timeoutSeconds(value) {
  const match = value.match(/^(\d+)(ms|s|m)?$/);
  if (!match) {
    throw new Error(`unsupported timeout value ${value}`);
  }

  const amount = Number(match[1]);
  const unit = match[2] ?? "s";
  if (unit === "ms") {
    return amount / 1000;
  }
  if (unit === "m") {
    return amount * 60;
  }
  return amount;
}

function directive(block, name) {
  const match = block.match(new RegExp(`(?:^|\\n)\\s*${name}\\s+([^;]+);`));
  if (!match) {
    throw new Error(`missing ${name}`);
  }
  return match[1].trim();
}

function assertMinTimeout(block, name, minimumSeconds) {
  const seconds = timeoutSeconds(directive(block, name));
  if (seconds < minimumSeconds) {
    throw new Error(`${name} must be at least ${minimumSeconds}s, got ${seconds}s`);
  }
}

for (const pattern of ["location = /api/v1", "location /api/v1/"]) {
  const block = locationBlock(pattern);
  assertMinTimeout(block, "proxy_read_timeout", 600);
  assertMinTimeout(block, "proxy_send_timeout", 600);
  assertMinTimeout(block, "send_timeout", 600);
}

console.log("nginx timeout tests passed");
