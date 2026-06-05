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

export function formatRate(value: number) {
  return `${formatBytes(value)}/s`;
}

export function formatTimestamp(value: number | null | undefined) {
  if (!value) {
    return "unknown";
  }
  return new Date(value).toLocaleString();
}

export function formatDurationSince(value: number | null | undefined) {
  if (!value) {
    return "unknown";
  }
  const elapsed = Math.max(0, Date.now() - value);
  const seconds = Math.floor(elapsed / 1000);
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const remainingSeconds = seconds % 60;
  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }
  if (minutes > 0) {
    return `${minutes}m ${remainingSeconds}s`;
  }
  return `${remainingSeconds}s`;
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
