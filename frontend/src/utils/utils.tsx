import type { Threshold } from "../types/Threshold";

export function capitaliseFirstLetter(str: string): string {
    if (!str) return str;
    return str.charAt(0).toUpperCase() + str.slice(1);
}

export function formatNumber(n: number): string {
    // round off the numbers into 3sf and append number suffixes at the end
    if (n >= 1_000_000_000) return `${(n / 1_000_000_000).toPrecision(3)}B`;
    if (n >= 1_000_000) return `${(n / 1_000_000).toPrecision(3)}M`;
    if (n >= 1_000) return `${(n / 1_000).toPrecision(3)}k`;
    return n.toString();
}

export function formatInterval(ms: number): string {
    if (ms < 60_000) return `${Math.round(ms / 1000)}s`;
    return `${Math.round(ms / 60_000)}m`;
}

export const DEFAULT_THRESHOLDS: Threshold[] = [
    { min: 80, colour: "#ef5350", label: "Critical" },
    { min: 60, colour: "#ffa726", label: "High" },
    { min: 40, colour: "#fbc02d", label: "Medium" },
    { min: 20, colour: "#42a5f5", label: "Low" },
    { min: 0, colour: "#607d8b", label: "Minimal" },
];

export const STORAGE_KEY = "serviceErrorThresholds";
export const HEX_COLOUR_REGEX = /^#[0-9a-fA-F]{6}$/;

export function loadThresholds(): Threshold[] {
    try {
        const saved = localStorage.getItem(STORAGE_KEY);
        if (saved) {
            const parsed = JSON.parse(saved);
            // valid threshold only if min is a number between 0 to 100 inclusive,
            // label is a string and valid hex code for colour
            const isValid =
                Array.isArray(parsed) &&
                parsed.every(
                    (t) =>
                        typeof t.min === "number" &&
                        Number.isInteger(t.min) &&
                        t.min >= 0 &&
                        t.min <= 100 &&
                        typeof t.label === "string" &&
                        t.label.trim().length > 0 &&
                        HEX_COLOUR_REGEX.test(t.colour),
                );
            if (isValid) return parsed;
            console.warn("Stored thresholds failed validation, using defaults");
        }
    } catch (err) {
        console.warn("Failed to load thresholds, using defaults", err);
    }
    return DEFAULT_THRESHOLDS;
}

export function saveThresholds(thresholds: Threshold[]): void {
    try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(thresholds));
    } catch (err) {
        console.warn("Failed to persist thresholds", err);
    }
}

export function getSeverityColour(errorRate: number, thresholds: Threshold[]): string {
    const sorted = [...thresholds].sort((a, b) => b.min - a.min);
    return sorted.find(({ min }) => errorRate >= min)?.colour ?? "#607d8b";
}

export const formatTime = (v: number) => new Date(v).toLocaleTimeString();
