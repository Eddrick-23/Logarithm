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
