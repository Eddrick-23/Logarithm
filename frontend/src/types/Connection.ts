export type ConnectionStatus = "connecting" | "live" | "error";

export const STATUS_CONFIG = {
    live: { label: "LIVE", colour: "success.main" },
    connecting: { label: "CONNECTING", colour: "warning.main" },
    error: { label: "OFFLINE", colour: "error.main" },
};
