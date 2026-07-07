import type { LogType } from "../types/Log";

const SEVERITY_NORMALISE_MAP: Record<string, LogType> = {
    trace: "trace",
    debug: "debug",
    info: "info",
    warning: "warn", // logs coming in have severity text of WARNING
    warn: "warn", // by default OTEL uses "warn" instead of "warning" but we support both
    error: "error",
    fatal: "fatal",
};

export const severityStyles: Record<LogType, { bg: string; text: string }> = {
    trace: { bg: "rgba(189, 189, 189, 0.15)", text: "#bdbdbd" }, // Grey
    debug: { bg: "rgba(100, 181, 246, 0.15)", text: "#64b5f6" }, // Blue
    info: { bg: "rgba(102, 187, 106, 0.15)", text: "success.main" }, // Green
    warn: { bg: "rgba(255, 167, 38, 0.15)", text: "warning.main" }, // Orange
    error: { bg: "rgba(239, 83, 80, 0.15)", text: "error.main" }, // Red
    fatal: { bg: "rgba(171, 71, 188, 0.15)", text: "#ab47bc" }, // Purple
};

export const parseSeverity = (severityText: string | null | undefined): LogType => {
    // guard in case log has no severity text
    if (!severityText) return "info";
    // strip numbered variants: "DEBUG2" -> "debug", "WARN3" -> "warn"
    const base = severityText.replace(/\d+$/, "").toLowerCase();
    // fallback to info if we get an unsupported severity name
    return SEVERITY_NORMALISE_MAP[base] ?? "info";
};
