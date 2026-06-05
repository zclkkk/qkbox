import type { StructuredError } from "./api/client";

export function formatBytes(value: number) {
  if (value < 1024) {
    return `${value} B`;
  }
  if (value < 1024 * 1024) {
    return `${(value / 1024).toFixed(1)} KiB`;
  }
  return `${(value / 1024 / 1024).toFixed(1)} MiB`;
}

export function formatError(error: StructuredError | Error | string | null | undefined) {
  if (!error) {
    return null;
  }
  if (typeof error === "string") {
    return error;
  }
  if ("code" in error && "message" in error) {
    return `${error.code}: ${error.message}`;
  }
  return error.message;
}
