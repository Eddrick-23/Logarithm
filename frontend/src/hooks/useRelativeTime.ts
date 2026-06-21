import { useTick } from "./useTick";

/**
 * Returns a live "Xs ago" / "Xm ago" string for a given timestamp (ms),
 * ticking once a second. Returns null if no timestamp is provided.
 */
export function useRelativeTime(timestamp?: number, justNowThreshold = 2) {
    useTick();

    if (!timestamp) return null;
    const diffSec = Math.max(0, Math.round((Date.now() - timestamp) / 1000));
    if (diffSec < justNowThreshold) return "just now";
    if (diffSec < 60) return `${diffSec}s ago`;
    const diffMin = Math.round(diffSec / 60);
    if (diffMin < 60) return `${diffMin}m ago`;
    return `${Math.round(diffMin / 60)}h ago`;
}
